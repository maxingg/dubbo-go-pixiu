//go:build integ
// +build integ

// Copyright Istio Authors
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

package cni

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/dubbo-go-pixiu/pkg/test/env"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/components/echo/common/deployment"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/components/istio"
	"github.com/apache/dubbo-go-pixiu/pkg/test/framework/label"
	"github.com/apache/dubbo-go-pixiu/pkg/test/kube"
	"github.com/apache/dubbo-go-pixiu/pkg/test/util/file"
	"github.com/apache/dubbo-go-pixiu/pkg/test/util/retry"
	"github.com/apache/dubbo-go-pixiu/tests/integration/pilot/common"
)

var (
	i istio.Instance

	apps = deployment.SingleNamespaceView{}
)

const (
	// TODO: replace this with official 1.11 release once available.
	NMinusOne    = "1.11.0-beta.1"
	CNIConfigDir = "tests/integration/pilot/testdata/upgrade"
)

// Currently only test CNI with one version behind.
var versions = []string{NMinusOne}

// TestCNIVersionSkew runs all traffic tests with older versions of CNI and lastest Istio.
// This is to simulate the case where CNI and Istio control plane versions are out of sync during upgrade.
func TestCNIVersionSkew(t *testing.T) {
	framework.
		NewTest(t).
		Features("traffic.cni.upgrade").
		Run(func(t framework.TestContext) {
			if !i.Settings().EnableCNI {
				t.Skip("CNI version skew test is only tested when CNI is enabled.")
			}
			for _, v := range versions {
				installCNIOrFail(t, v)
				podFetchFn := kube.NewSinglePodFetch(t.Clusters().Default(), "kube-system", "k8s-app=istio-cni-node")
				// Make sure CNI pod is using image with applied version.
				retry.UntilSuccessOrFail(t, func() error {
					pods, err := podFetchFn()
					if err != nil {
						return fmt.Errorf("failed to get CNI pods %v", err)
					}
					if len(pods) == 0 {
						return fmt.Errorf("cannot find any CNI pods")
					}
					for _, p := range pods {
						if !strings.Contains(p.Spec.Containers[0].Image, v) {
							return fmt.Errorf("pods image does not match wanted CNI version")
						}
					}
					return nil
				})

				// Make sure CNI pod is ready
				if _, err := kube.WaitUntilPodsAreReady(podFetchFn); err != nil {
					t.Fatal(err)
				}
				if err := apps.All.Instances().Restart(); err != nil {
					t.Fatalf("Failed to restart apps %v", err)
				}
				common.RunAllTrafficTests(t, i, &apps)
			}
		})
}

func TestMain(m *testing.M) {
	// nolint: staticcheck
	framework.
		NewSuite(m).
		Label(label.Postsubmit).
		Label(label.CustomSetup).
		RequireMultiPrimary().
		Setup(istio.Setup(&i, nil)).
		Setup(deployment.SetupSingleNamespace(&apps)).
		Run()
}

// installCNIOrFail installs CNI DaemonSet for the given version.
// It looks for tar compressed CNI manifest and apply that in the cluster.
func installCNIOrFail(t framework.TestContext, ver string) {
	cniFilePath := filepath.Join(env.IstioSrc, CNIConfigDir,
		fmt.Sprintf("%s-cni-install.yaml.tar", ver))
	config, err := file.ReadTarFile(cniFilePath)
	if err != nil {
		t.Fatalf("Failed to read CNI manifest %v", err)
	}
	t.ConfigIstio().YAML("", config).ApplyOrFail(t)
}
