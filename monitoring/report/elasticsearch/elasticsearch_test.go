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

package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/njcx/libbeat_v7/monitoring/report"
)

func TestMakeClientParams(t *testing.T) {
	tests := map[string]struct {
		format   report.Format
		params   map[string]string
		expected map[string]string
	}{
		"format_bulk": {
			report.FormatBulk,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "bar",
			},
		},
		"format_xpack_monitoring_bulk": {
			report.FormatXPackMonitoringBulk,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo":                "bar",
				"system_id":          "beats",
				"system_api_version": "7",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			params := makeClientParams(config{
				Format: test.format,
				Params: test.params,
			})

			require.Equal(t, test.expected, params)
		})
	}
}
