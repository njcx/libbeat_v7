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

//go:build integration
// +build integration

package mlimporter

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/njcx/libbeat_v7/common/transport/httpcommon"
	"github.com/njcx/libbeat_v7/esleg/eslegclient"
	"github.com/njcx/libbeat_v7/esleg/eslegtest"
	"github.com/njcx/libbeat_v7/logp"
)

const sampleJob = `
{
  "description" : "Anomaly detector for changes in event rates of nginx.access.response_code responses",
  "analysis_config" : {
    "bucket_span": "1h",
    "summary_count_field_name": "doc_count",
    "detectors": [
      {
        "detector_description": "Event rate for nginx.access.response_code",
        "function": "count",
        "partition_field_name": "nginx.access.response_code"
      }
    ],
    "influencers": ["nginx.access.response_code"]
  },
  "data_description": {
    "time_field": "@timestamp",
    "time_format": "epoch_ms"
  },
  "model_plot_config": {
    "enabled": true
  }
}
`

const sampleDatafeed = `
{
    "job_id": "PLACEHOLDER",
    "indexes": [
      "filebeat-*"
    ],
    "types": [
      "doc",
      "log"
    ],
    "query": {
      "match_all": {
        "boost": 1
      }
    },
    "aggregations": {
      "buckets": {
        "date_histogram": {
          "field": "@timestamp",
          "interval": 3600000,
          "offset": 0,
          "order": {
            "_key": "asc"
          },
          "keyed": false,
          "min_doc_count": 0
        },
        "aggregations": {
          "@timestamp": {
            "max": {
              "field": "@timestamp"
            }
          },
          "nginx.access.response_code": {
              "terms": {
                "field": "nginx.access.response_code",
                "size": 10000
              }
          }
        }
      }
    }
}
`

func TestImportJobs(t *testing.T) {
	logp.TestingSetup()

	client := getTestingElasticsearch(t)

	haveXpack, err := HaveXpackML(client)
	assert.NoError(t, err)
	if !haveXpack {
		t.Skip("Skip ML tests because xpack/ML is not available in Elasticsearch")
	}

	workingDir, err := ioutil.TempDir("", "machine-learning")
	assert.NoError(t, err)
	defer os.RemoveAll(workingDir)

	assert.NoError(t, ioutil.WriteFile(workingDir+"/job.json", []byte(sampleJob), 0644))
	assert.NoError(t, ioutil.WriteFile(workingDir+"/datafeed.json", []byte(sampleDatafeed), 0644))

	mlconfig := MLConfig{
		ID:           "test-ml-config",
		JobPath:      workingDir + "/job.json",
		DatafeedPath: workingDir + "/datafeed.json",
	}

	err = ImportMachineLearningJob(client, &mlconfig)
	assert.NoError(t, err)

	var mlBaseURL string
	if client.GetVersion().Major < 7 {
		mlBaseURL = "/_xpack/ml"
	} else {
		mlBaseURL = "/_ml"
	}

	// check by GETing back
	status, response, err := client.Request("GET", mlBaseURL+"/anomaly_detectors", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	logp.Debug("mltest", "Response: %s", response)

	type jobRes struct {
		Count int `json:"count"`
		Jobs  []struct {
			JobId   string `json:"job_id"`
			JobType string `json:"job_type"`
		}
	}
	var res jobRes

	err = json.Unmarshal(response, &res)
	assert.NoError(t, err)
	assert.True(t, res.Count >= 1)
	found := false
	for _, job := range res.Jobs {
		if job.JobId == "test-ml-config" {
			found = true
			assert.Equal(t, job.JobType, "anomaly_detector")
		}
	}
	assert.True(t, found)

	status, response, err = client.Request("GET", mlBaseURL+"/datafeeds", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	logp.Debug("mltest", "Response: %s", response)
	type datafeedRes struct {
		Count     int `json:"count"`
		Datafeeds []struct {
			DatafeedId string `json:"datafeed_id"`
			JobId      string `json:"job_id"`
			QueryDelay string `json:"query_delay"`
		}
	}
	var df datafeedRes
	err = json.Unmarshal(response, &df)
	assert.NoError(t, err)
	assert.True(t, df.Count >= 1)
	found = false
	for _, datafeed := range df.Datafeeds {
		if datafeed.DatafeedId == "datafeed-test-ml-config" {
			found = true
			assert.Equal(t, datafeed.JobId, "test-ml-config")
			assert.Equal(t, datafeed.QueryDelay, "87034ms")
		}
	}
	assert.True(t, found)

	// importing again should not error out
	err = ImportMachineLearningJob(client, &mlconfig)
	assert.NoError(t, err)
}

func getTestingElasticsearch(t eslegtest.TestLogger) *eslegclient.Connection {
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL: eslegtest.GetURL(),
		Transport: httpcommon.HTTPTransportSettings{
			Timeout: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	conn.Encoder = eslegclient.NewJSONEncoder(nil, false)

	err = conn.Connect()
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	return conn
}
