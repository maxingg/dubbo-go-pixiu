/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package traffic

import (
	"math/rand"
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/common/extension/filter"
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/context/http"
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/logger"
)

const (
	canaryByHeader CanaryHeaders = "canary-by-header"
	unInitialize   int           = -1
)

func init() {
	filter.RegisterHttpFilter(&Plugin{})
}

type (
	CanaryHeaders string

	Plugin struct {
	}

	FilterFactory struct {
		cfg *Config
	}

	Filter struct {
		weight int
		record map[string]struct{}
		Rules  []*ClusterWrapper
	}

	Config struct {
		Traffics []*Cluster `yaml:"traffics" json:"traffics" mapstructure:"traffics"`
	}

	ClusterWrapper struct {
		Cluster     *Cluster
		header      string
		weightCeil  int
		weightFloor int
	}

	Cluster struct {
		Name           string `yaml:"name" json:"name" mapstructure:"name"`
		Router         string `yaml:"router" json:"router" mapstructure:"router"`
		CanaryByHeader string `yaml:"canary-by-header" json:"canary-by-header" mapstructure:"canary-by-header"`
		CanaryWeight   int    `yaml:"canary-weight" json:"canary-weight" mapstructure:"canary-weight"`
	}
)

func (p *Plugin) Kind() string {
	return constant.HTTPTrafficFilter
}

func (p *Plugin) CreateFilterFactory() (filter.HttpFilterFactory, error) {
	return &FilterFactory{
		cfg: &Config{},
	}, nil
}

func (factory *FilterFactory) Config() interface{} {
	return factory.cfg
}

func (factory *FilterFactory) Apply() error {
	return nil
}

func (factory *FilterFactory) PrepareFilterChain(ctx *http.HttpContext, chain filter.FilterChain) error {
	f := &Filter{
		weight: unInitialize,
		record: map[string]struct{}{},
	}
	f.Rules = factory.rulesMatch(f, ctx.Request.RequestURI)
	chain.AppendDecodeFilters(f)
	return nil
}

func (f *Filter) Decode(ctx *http.HttpContext) filter.FilterStatus {
	cluster := ""
	if f.Rules != nil {
		for _, wp := range f.Rules {
			if f.trafficHeader(wp, ctx) {
				cluster = wp.Cluster.Name
				logger.Debugf("[dubbo-go-pixiu] execute traffic split to cluster %s", wp.Cluster.Name)
				break
			}
		}
		if cluster == "" {
			for _, wp := range f.Rules {
				if f.trafficWeight(wp, ctx) {
					ctx.Route.Cluster = wp.Cluster.Name
					cluster = wp.Cluster.Name
					logger.Debugf("[dubbo-go-pixiu] execute traffic split to cluster %s", wp.Cluster.Name)
					break
				}
			}
		}
		if cluster != "" {
			ctx.Route.Cluster = cluster
		}
	} else {
		logger.Warnf("[dubbo-go-pixiu] execute traffic split fail because of empty rules.")
	}
	return filter.Continue
}

func (f *Filter) trafficHeader(c *ClusterWrapper, ctx *http.HttpContext) bool {
	return spiltHeader(ctx.Request, c.header)
}

func (f *Filter) trafficWeight(c *ClusterWrapper, ctx *http.HttpContext) bool {
	if f.weight == unInitialize {
		rand.Seed(time.Now().UnixNano())
		f.weight = rand.Intn(100) + 1
	}

	return spiltWeight(f.weight, c.weightFloor, c.weightCeil)
}

func (factory *FilterFactory) rulesMatch(f *Filter, path string) []*ClusterWrapper {
	clusters := factory.cfg.Traffics

	if clusters != nil {
		rules := make([]*ClusterWrapper, 0)
		up := 0
		for _, cluster := range clusters {
			if strings.HasPrefix(path, cluster.Router) {
				wp := &ClusterWrapper{
					Cluster:     cluster,
					header:      "",
					weightCeil:  -1,
					weightFloor: -1,
				}
				if cluster.CanaryByHeader != "" {
					if _, ok := f.record[cluster.CanaryByHeader]; ok {
						logger.Errorf("Duplicate canary-by-header values")
					} else {
						f.record[cluster.CanaryByHeader] = struct{}{}
						wp.header = cluster.CanaryByHeader
					}
				}
				if cluster.CanaryWeight > 0 && cluster.CanaryWeight <= 100 {
					wp.weightFloor = up
					up += cluster.CanaryWeight
					if up > 100 {
						logger.Errorf("[dubbo-go-pixiu] clusters' weight sum more than 100 in %v service!", cluster.Router)
						continue
					}
					wp.weightCeil = up
				}
				rules = append(rules, wp)
			}
		}
		return rules
	}
	return nil
}
