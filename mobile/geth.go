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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gec

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ecchain/go-ecchain/core"
	"github.com/ecchain/go-ecchain/ec"
	"github.com/ecchain/go-ecchain/ec/downloader"
	"github.com/ecchain/go-ecchain/ecclient"
	"github.com/ecchain/go-ecchain/ecstats"
	"github.com/ecchain/go-ecchain/les"
	"github.com/ecchain/go-ecchain/node"
	"github.com/ecchain/go-ecchain/p2p"
	"github.com/ecchain/go-ecchain/p2p/nat"
	"github.com/ecchain/go-ecchain/params"
	whisper "github.com/ecchain/go-ecchain/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Gec
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-ecereum to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Booecrap nodes used to establish connectivity with the rest of the network.
	BooecrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// ecchainEnabled specifies whecer the node should run the ecchain protocol.
	ecchainEnabled bool

	// ecchainNetworkID is the network identifier used by the ecchain protocol to
	// decide if remote peers should be accepted or not.
	ecchainNetworkID int64 // uint64 in truth, but Java can't handle that...

	// ecchainGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	ecchainGenesis string

	// ecchainDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	ecchainDatabaseCache int

	// ecchainNeecats is a neecats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	ecchainNeecats string

	// WhisperEnabled specifies whecer the node should run the Whisper protocol.
	WhisperEnabled bool
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BooecrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	ecchainEnabled:       true,
	ecchainNetworkID:     1,
	ecchainDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gec ecchain node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gec node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BooecrapNodes == nil || config.BooecrapNodes.Size() == 0 {
		config.BooecrapNodes = defaultNodeConfig.BooecrapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BooecrapNodesV5: config.BooecrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.ecchainGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.ecchainGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.ecchainGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.ecchainNetworkID == 1 {
				config.ecchainNetworkID = 3
			}
		}
	}
	// Register the ecchain protocol if requested
	if config.ecchainEnabled {
		ecConf := ec.DefaultConfig
		ecConf.Genesis = genesis
		ecConf.SyncMode = downloader.LightSync
		ecConf.NetworkId = uint64(config.ecchainNetworkID)
		ecConf.DatabaseCache = config.ecchainDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &ecConf)
		}); err != nil {
			return nil, fmt.Errorf("ecereum init: %v", err)
		}
		// If neecats reporting is requested, do it
		if config.ecchainNeecats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.Lightecchain
				ctx.Service(&lesServ)

				return ecstats.New(config.ecchainNeecats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("neecats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetecchainClient retrieves a client to access the ecchain subsystem.
func (n *Node) GetecchainClient() (client *ecchainClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &ecchainClient{ecclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
