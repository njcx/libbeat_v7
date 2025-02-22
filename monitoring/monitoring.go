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

package monitoring

import (
	"errors"

	"github.com/njcx/libbeat_v7/common"
	"github.com/njcx/libbeat_v7/common/cfgwarn"
	"github.com/njcx/libbeat_v7/monitoring/report"
)

// BeatConfig represents the part of the $BEAT.yml to do with monitoring settings
type BeatConfig struct {
	XPackMonitoring *common.Config `config:"xpack.monitoring"`
	Monitoring      *common.Config `config:"monitoring"`
}

type Mode uint8

//go:generate stringer -type=Mode
const (
	// Reported mode, is lowest report level with most basic metrics only
	Reported Mode = iota

	// Full reports all metrics
	Full
)

var (
	errMonitoringBothConfigEnabled = errors.New("both xpack.monitoring.* and monitoring.* cannot be set. Prefer to set monitoring.* and set monitoring.elasticsearch.hosts to monitoring cluster hosts")
	warnMonitoringDeprecatedConfig = "xpack.monitoring.* settings are deprecated. Use monitoring.* instead, but set monitoring.elasticsearch.hosts to monitoring cluster hosts."
)

// Default is the global default metrics registry provided by the monitoring package.
var Default = NewRegistry()

func init() {
	GetNamespace("stats").SetRegistry(Default)
}

var errNotFound = errors.New("Name unknown")
var errInvalidName = errors.New("Name does not point to a valid variable")

func VisitMode(mode Mode, vs Visitor) {
	Default.Visit(mode, vs)
}

func Visit(vs Visitor) {
	Default.Visit(Full, vs)
}

func Do(mode Mode, f func(string, interface{})) {
	Default.Do(mode, f)
}

func Get(name string) Var {
	return Default.Get(name)
}

func GetRegistry(name string) *Registry {
	return Default.GetRegistry(name)
}

func Remove(name string) {
	Default.Remove(name)
}

func Clear() error {
	return Default.Clear()
}

// SelectConfig selects the appropriate monitoring configuration based on the user's settings in $BEAT.yml. Users may either
// use xpack.monitoring.* settings OR monitoring.* settings but not both.
func SelectConfig(beatCfg BeatConfig) (*common.Config, *report.Settings, error) {
	switch {
	case beatCfg.Monitoring.Enabled() && beatCfg.XPackMonitoring.Enabled():
		return nil, nil, errMonitoringBothConfigEnabled
	case beatCfg.XPackMonitoring.Enabled():
		cfgwarn.Deprecate("8.0.0", warnMonitoringDeprecatedConfig)
		monitoringCfg := beatCfg.XPackMonitoring
		return monitoringCfg, &report.Settings{Format: report.FormatXPackMonitoringBulk}, nil
	case beatCfg.Monitoring.Enabled():
		monitoringCfg := beatCfg.Monitoring
		return monitoringCfg, &report.Settings{Format: report.FormatBulk}, nil
	default:
		return nil, nil, nil
	}
}

// GetClusterUUID returns the value of the monitoring.cluster_uuid setting, if it is set.
func GetClusterUUID(monitoringCfg *common.Config) (string, error) {
	if monitoringCfg == nil {
		return "", nil
	}

	var config struct {
		ClusterUUID string `config:"cluster_uuid"`
	}
	if err := monitoringCfg.Unpack(&config); err != nil {
		return "", err
	}

	return config.ClusterUUID, nil
}

// IsEnabled returns whether the monitoring reporter is enabled or not.
func IsEnabled(monitoringCfg *common.Config) bool {
	if monitoringCfg == nil {
		return false
	}

	// If the only setting in the monitoring config is cluster_uuid, it is
	// not enabled
	fields := monitoringCfg.GetFields()
	if len(fields) == 1 && fields[0] == "cluster_uuid" {
		return false
	}

	return monitoringCfg.Enabled()
}
