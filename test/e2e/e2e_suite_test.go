//go:build e2e

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	e2eCtx *TestContext
)

func init() {
	e2eCtx = &TestContext{
		Settings: Settings{
			ConfigPath:     "config/ironcore.yaml",
			ArtifactFolder: "_artifacts",
			DataFolder:     "data",
			SkipCleanup:    false,
		},
	}

	flag.StringVar(&e2eCtx.Settings.ConfigPath, "e2e.config",
		e2eCtx.Settings.ConfigPath, "path to the e2e config file")
	flag.BoolVar(&e2eCtx.Settings.SkipCleanup, "e2e.skip-resource-cleanup",
		e2eCtx.Settings.SkipCleanup, "if true, the resource cleanup after tests will be skipped")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	ctrl.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.WriteTo(GinkgoWriter)))
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	Expect(e2eCtx.Settings.ConfigPath).To(BeAnExistingFile(), "e2e-config path must be valid")
	e2eCtx.E2EConfig = clusterctl.LoadE2EConfig(context.TODO(), clusterctl.LoadE2EConfigInput{
		ConfigPath: e2eCtx.Settings.ConfigPath,
	})

	var err error
	e2eCtx.Settings.ArtifactFolder, err = filepath.Abs(e2eCtx.Settings.ArtifactFolder)
	Expect(err).ToNot(HaveOccurred(), "Failed to resolve absolute path for artifacts")
	if e2eCtx.Settings.ArtifactFolder != "" {
		Expect(os.MkdirAll(e2eCtx.Settings.ArtifactFolder, 0755)).To(Succeed())
	}

	e2eCtx.Environment.ClusterctlConfigPath = clusterctl.CreateRepository(context.TODO(), clusterctl.CreateRepositoryInput{
		E2EConfig:        e2eCtx.E2EConfig,
		RepositoryFolder: filepath.Join(e2eCtx.Settings.ArtifactFolder, "repository"),
	})
	Expect(e2eCtx.Environment.ClusterctlConfigPath).To(BeAnExistingFile(), "Failed to create clusterctl config")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1.AddToScheme(scheme))
	e2eCtx.Environment.Scheme = scheme
})

var _ = AfterSuite(func() {

})
