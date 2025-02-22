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

package transport

import (
	"time"

	"github.com/njcx/libbeat_v7/common/transport"

	"github.com/njcx/libbeat_v7/testing"
)

func NetDialer(timeout time.Duration) Dialer {
	return transport.NetDialer(timeout)
}

func TestNetDialer(d testing.Driver, timeout time.Duration) Dialer {
	return transport.TestNetDialer(d, timeout)
}

func UnixDialer(timeout time.Duration, sockFile string) Dialer {
	return transport.UnixDialer(timeout, sockFile)
}

func TestUnixDialer(d testing.Driver, timeout time.Duration, sockFile string) Dialer {
	return transport.TestUnixDialer(d, timeout, sockFile)
}
