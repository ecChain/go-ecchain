// Copyright 2015 The go-ethereum Authors
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

package ec

import (
	"context"
	"math/big"

	"github.com/ecchain/go-ecchain/accounts"
	"github.com/ecchain/go-ecchain/common"
	"github.com/ecchain/go-ecchain/common/math"
	"github.com/ecchain/go-ecchain/core"
	"github.com/ecchain/go-ecchain/core/bloombits"
	"github.com/ecchain/go-ecchain/core/state"
	"github.com/ecchain/go-ecchain/core/types"
	"github.com/ecchain/go-ecchain/core/vm"
	"github.com/ecchain/go-ecchain/ec/downloader"
	"github.com/ecchain/go-ecchain/ec/gasprice"
	"github.com/ecchain/go-ecchain/ecdb"
	"github.com/ecchain/go-ecchain/event"
	"github.com/ecchain/go-ecchain/params"
	"github.com/ecchain/go-ecchain/rpc"
)

// ecApiBackend implements ethapi.Backend for full nodes
type ecApiBackend struct {
	ec *ecchain
	gpo *gasprice.Oracle
}

func (b *ecApiBackend) ChainConfig() *params.ChainConfig {
	return b.ec.chainConfig
}

func (b *ecApiBackend) CurrentBlock() *types.Block {
	return b.ec.blockchain.CurrentBlock()
}

func (b *ecApiBackend) SetHead(number uint64) {
	b.ec.protocolManager.downloader.Cancel()
	b.ec.blockchain.SetHead(number)
}

func (b *ecApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.ec.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ec.blockchain.CurrentBlock().Header(), nil
	}
	return b.ec.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *ecApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.ec.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ec.blockchain.CurrentBlock(), nil
	}
	return b.ec.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *ecApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.ec.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.ec.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *ecApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.ec.blockchain.GetBlockByHash(blockHash), nil
}

func (b *ecApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.ec.chainDb, blockHash, core.GetBlockNumber(b.ec.chainDb, blockHash)), nil
}

func (b *ecApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := core.GetBlockReceipts(b.ec.chainDb, blockHash, core.GetBlockNumber(b.ec.chainDb, blockHash))
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *ecApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.ec.blockchain.GetTdByHash(blockHash)
}

func (b *ecApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.ec.BlockChain(), nil)
	return vm.NewEVM(context, state, b.ec.chainConfig, vmCfg), vmError, nil
}

func (b *ecApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.ec.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *ecApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.ec.BlockChain().SubscribeChainEvent(ch)
}

func (b *ecApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.ec.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *ecApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.ec.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *ecApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.ec.BlockChain().SubscribeLogsEvent(ch)
}

func (b *ecApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.ec.txPool.AddLocal(signedTx)
}

func (b *ecApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.ec.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *ecApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.ec.txPool.Get(hash)
}

func (b *ecApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.ec.txPool.State().GetNonce(addr), nil
}

func (b *ecApiBackend) Stats() (pending int, queued int) {
	return b.ec.txPool.Stats()
}

func (b *ecApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.ec.TxPool().Content()
}

func (b *ecApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.ec.TxPool().SubscribeTxPreEvent(ch)
}

func (b *ecApiBackend) Downloader() *downloader.Downloader {
	return b.ec.Downloader()
}

func (b *ecApiBackend) ProtocolVersion() int {
	return b.ec.ecVersion()
}

func (b *ecApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *ecApiBackend) ChainDb() ecdb.Database {
	return b.ec.ChainDb()
}

func (b *ecApiBackend) EventMux() *event.TypeMux {
	return b.ec.EventMux()
}

func (b *ecApiBackend) AccountManager() *accounts.Manager {
	return b.ec.AccountManager()
}

func (b *ecApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.ec.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *ecApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.ec.bloomRequests)
	}
}
