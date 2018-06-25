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

package ecclient

import "github.com/ecchain/go-ecchain"

// Verify that Client implements the ecereum interfaces.
var (
	_ = ecereum.ChainReader(&Client{})
	_ = ecereum.TransactionReader(&Client{})
	_ = ecereum.ChainStateReader(&Client{})
	_ = ecereum.ChainSyncReader(&Client{})
	_ = ecereum.ContractCaller(&Client{})
	_ = ecereum.GasEstimator(&Client{})
	_ = ecereum.GasPricer(&Client{})
	_ = ecereum.LogFilterer(&Client{})
	_ = ecereum.PendingStateReader(&Client{})
	// _ = ecereum.PendingStateEventer(&Client{})
	_ = ecereum.PendingContractCaller(&Client{})
)
