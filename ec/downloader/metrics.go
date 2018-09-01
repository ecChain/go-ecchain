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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ecchain/go-ecchain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("ec/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("ec/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("ec/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("ec/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("ec/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("ec/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("ec/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("ec/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("ec/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("ec/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("ec/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("ec/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("ec/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("ec/downloader/states/drop", nil)
)
