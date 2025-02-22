// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package spool

import (
	"github.com/njcx/libbeat_v7/publisher"
)

// producer -> broker API
type (
	pushRequest struct {
		event publisher.Event
		seq   uint32
		state *produceState
	}

	producerCancelRequest struct {
		state *produceState
		resp  chan producerCancelResponse
	}

	producerCancelResponse struct {
		removed int
	}
)

// consumer -> broker API

type (
	getRequest struct {
		sz   int              // request sz events from the broker
		resp chan getResponse // channel to send response to
	}

	getResponse struct {
		ack chan batchAckMsg
		err error
		buf []publisher.Event
	}

	batchAckMsg struct{}

	batchCancelRequest struct {
		// ack *ackChan
	}
)
