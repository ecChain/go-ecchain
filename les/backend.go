// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ecereum library.
//
// The go-ecereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ecereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ecereum library. If not, see <http://www.gnu.org/licenses/>.

// Package les implements the Light ecchain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ecchain/go-ecchain/accounts"
	"github.com/ecchain/go-ecchain/common"
	"github.com/ecchain/go-ecchain/common/hexutil"
	"github.com/ecchain/go-ecchain/consensus"
	"github.com/ecchain/go-ecchain/core"
	"github.com/ecchain/go-ecchain/core/bloombits"
	"github.com/ecchain/go-ecchain/core/types"
	"github.com/ecchain/go-ecchain/ec"
	"github.com/ecchain/go-ecchain/ec/downloader"
	"github.com/ecchain/go-ecchain/ec/filters"
	"github.com/ecchain/go-ecchain/ec/gasprice"
	"github.com/ecchain/go-ecchain/ecdb"
	"github.com/ecchain/go-ecchain/event"
	"github.com/ecchain/go-ecchain/internal/ethapi"
	"github.com/ecchain/go-ecchain/light"
	"github.com/ecchain/go-ecchain/log"
	"github.com/ecchain/go-ecchain/node"
	"github.com/ecchain/go-ecchain/p2p"
	"github.com/ecchain/go-ecchain/p2p/discv5"
	"github.com/ecchain/go-ecchain/params"
	rpc "github.com/ecchain/go-ecchain/rpc"
)

type Lightecchain struct {
	config *ec.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb ecdb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *ec.Config) (*Lightecchain, error) {
	chainDb, err := ec.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lec := &Lightecchain{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           ec.CreateConsensusEngine(ctx, &config.ecash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     ec.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lec.relay = NewLesTxRelay(peers, lec.reqDist)
	lec.serverPool = newServerPool(chainDb, quitSync, &lec.wg)
	lec.retriever = newRetrieveManager(peers, lec.reqDist, lec.serverPool)
	lec.odr = NewLesOdr(chainDb, lec.chtIndexer, lec.bloomTrieIndexer, lec.bloomIndexer, lec.retriever)
	if lec.blockchain, err = light.NewLightChain(lec.odr, lec.chainConfig, lec.engine); err != nil {
		return nil, err
	}
	lec.bloomIndexer.Start(lec.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lec.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lec.txPool = light.NewTxPool(lec.chainConfig, lec.blockchain, lec.relay)
	if lec.protocolManager, err = NewProtocolManager(lec.chainConfig, true, ClientProtocolVersions, config.NetworkId, lec.eventMux, lec.engine, lec.peers, lec.blockchain, nil, chainDb, lec.odr, lec.relay, quitSync, &lec.wg); err != nil {
		return nil, err
	}
	lec.ApiBackend = &LesApiBackend{lec, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lec.ApiBackend.gpo = gasprice.NewOracle(lec.ApiBackend, gpoParams)
	return lec, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// ecerbase is the address that mining rewards will be send to
func (s *LightDummyAPI) ecerbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for ecerbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the ecereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Lightecchain) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "ec",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "ec",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "ec",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Lightecchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Lightecchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *Lightecchain) TxPool() *light.TxPool              { return s.txPool }
func (s *Lightecchain) Engine() consensus.Engine           { return s.engine }
func (s *Lightecchain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Lightecchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Lightecchain) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Lightecchain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// ecchain protocol implementation.
func (s *Lightecchain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// ecchain protocol.
func (s *Lightecchain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
