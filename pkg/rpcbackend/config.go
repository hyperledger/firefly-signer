// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpcbackend

import (
	"context"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
)

const (
	// ConfigMaxRequestConcurrency the maximum number of concurrent JSON-RPC requests get processed at a time
	ConfigMaxConcurrentRequest = "maxConcurrentRequest"
	// ConfigBatchEnabled whether to enable batching JSON-RPC requests: https://www.jsonrpc.org/specification#batch
	ConfigBatchEnabled = "batch.enabled"
	// ConfigBatchSize when the amount of queued requests reaches this number, they will be batched and dispatched
	ConfigBatchSize = "batch.size"
	// ConfigBatchTimeout when the time since the first request was queued reaches this timeout, all requests in the queue will be batched and dispatched
	ConfigBatchTimeout = "batch.timeout"
	// ConfigBatchTimeout the maximum number of concurrent batch dispatching process
	ConfigBatchMaxDispatchConcurrency = "batch.dispatchConcurrency"
)

const (
	DefaultConfigBatchSize           = 500
	DefaultConfigTimeout             = "50ms"
	DefaultConfigDispatchConcurrency = 50
)

type RPCClientBatchOptions struct {
	Enabled                     bool
	BatchDispatcherContext      context.Context
	BatchTimeout                time.Duration
	BatchSize                   int
	BatchMaxDispatchConcurrency int
}

type RPCClientOptions struct {
	MaxConcurrentRequest int64
	BatchOptions         *RPCClientBatchOptions
}

func InitConfig(section config.Section) {
	section.AddKnownKey(ConfigBatchEnabled, false)
	section.AddKnownKey(ConfigMaxConcurrentRequest, 0)
	section.AddKnownKey(ConfigBatchSize, DefaultConfigBatchSize)
	section.AddKnownKey(ConfigBatchTimeout, DefaultConfigTimeout)
	section.AddKnownKey(ConfigBatchMaxDispatchConcurrency, DefaultConfigDispatchConcurrency)
}

func ReadConfig(batchDispatcherContext context.Context, section config.Section) RPCClientOptions {
	return RPCClientOptions{
		MaxConcurrentRequest: section.GetInt64(ConfigMaxConcurrentRequest),
		BatchOptions: &RPCClientBatchOptions{
			Enabled:                     section.GetBool(ConfigBatchEnabled),
			BatchTimeout:                section.GetDuration(ConfigBatchTimeout),
			BatchSize:                   section.GetInt(ConfigBatchSize),
			BatchMaxDispatchConcurrency: section.GetInt(ConfigBatchMaxDispatchConcurrency),
			BatchDispatcherContext:      batchDispatcherContext,
		},
	}
}
