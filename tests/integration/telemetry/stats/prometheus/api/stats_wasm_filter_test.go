//go:build integ
// +build integ

// Copyright Istio Authors. All Rights Reserved.
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

package telemetryapi

import (
	"testing"

	"github.com/apache/dubbo-go-pixiu/pkg/test/framework"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/components/istio"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/label"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/resource"
	common "github.com/apache/dubbo-go-pixiu/tests/integration/telemetry/stats/prometheus"
)

// TestTelemetryAPIStats verifies the stats filter could emit expected client and server side
// metrics when configured with the Telemetry API (with EnvoyFilters disabled)
// This test focuses on stats filter and metadata exchange filter could work coherently with
// proxy bootstrap config with Wasm runtime. To avoid flake, it does not verify correctness
// of metrics, which should be covered by integration test in proxy repo.
func TestTelemetryAPIStats(t *testing.T) {
	common.TestStatsFilter(t, "observability.telemetry.stats.prometheus.http.nullvm")
}

func TestMain(m *testing.M) {
	framework.NewSuite(m).
		Label(label.CustomSetup).
		Label(label.IPv4). // https://github.com/istio/istio/issues/35915
		Setup(istio.Setup(common.GetIstioInstance(), setupConfig)).
		Setup(func(ctx resource.Context) error {
			i, err := istio.Get(ctx)
			if err != nil {
				return err
			}
			return ctx.ConfigIstio().YAML(i.Settings().SystemNamespace, `
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: mesh-default
spec:
  metrics:
  - providers:
    - name: prometheus
`).Apply()
		}).
		Setup(common.TestSetup).
		Run()
}

func setupConfig(c resource.Context, cfg *istio.Config) {
	if cfg == nil {
		return
	}
	cfg.Values["telemetry.v2.enabled"] = "false"
}
