// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"os"

	"testing"

	e2eshared "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/test/e2e/shared"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api/test/framework"

	"sigs.k8s.io/cluster-api/test/framework/bootstrap"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	e2eCtx *e2eshared.E2EContext
)

func init() {

	e2eCtx = &e2eshared.E2EContext{

		Settings: e2eshared.Settings{

			ConfigPath: "config/ironcore.yaml",

			ArtifactFolder: "_artifacts",

			DataFolder: "data",

			SkipCleanup: false,
		},
	}

	flag.StringVar(&e2eCtx.Settings.ConfigPath, "e2e.config", e2eCtx.Settings.ConfigPath, "path to the e2e config file")

	flag.StringVar(&e2eCtx.Settings.ArtifactFolder, "e2e.artifacts-folder", e2eCtx.Settings.ArtifactFolder, "folder where e2e test artifacts should be stored")

	flag.StringVar(&e2eCtx.Settings.DataFolder, "e2e.data-folder", e2eCtx.Settings.DataFolder, "root folder for the data required by the tests")

	flag.BoolVar(&e2eCtx.Settings.SkipCleanup, "e2e.skip-resource-cleanup", e2eCtx.Settings.SkipCleanup, "if true, the resource cleanup after tests will be skipped")

}

func TestE2E(t *testing.T) {

	RegisterFailHandler(Fail)

	ctrl.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.WriteTo(GinkgoWriter)))

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting cluster-api-provider-ironcore-metal suite\n")

	RunSpecs(t, "e2e suite")

}

var _ = BeforeSuite(func() {

	Expect(e2eCtx.Settings.ConfigPath).To(BeAnExistingFile(), "e2e-config path must be valid")

	var err error
	e2eCtx.Settings.ArtifactFolder, err = filepath.Abs(e2eCtx.Settings.ArtifactFolder)
	Expect(err).ToNot(HaveOccurred(), "Failed to resolve absolute path for artifacts")

	if e2eCtx.Settings.ArtifactFolder != "" {

		Expect(os.MkdirAll(e2eCtx.Settings.ArtifactFolder, 0755)).To(Succeed())

	}

	e2eCtx.E2EConfig = clusterctl.LoadE2EConfig(context.TODO(), clusterctl.LoadE2EConfigInput{

		ConfigPath: e2eCtx.Settings.ConfigPath,
	})

	e2eCtx.Environment.Scheme = e2eshared.CreateScheme()

	bootstrapClusterProvider := bootstrap.NewKindClusterProvider("e2e-bootstrap")

	bootstrapClusterProvider.Create(context.TODO())

	e2eCtx.Environment.BootstrapClusterProxy = framework.NewClusterProxy(

		"e2e-bootstrap",

		bootstrapClusterProvider.GetKubeconfigPath(),

		e2eCtx.Environment.Scheme,
	)

	//e2eCtx.Environment.ClusterctlConfigPath = e2eCtx.Settings.ConfigPath
	e2eCtx.Environment.ClusterctlConfigPath = clusterctl.CreateRepository(context.TODO(), clusterctl.CreateRepositoryInput{
		E2EConfig:        e2eCtx.E2EConfig,
		RepositoryFolder: filepath.Join(e2eCtx.Settings.ArtifactFolder, "repository"),
	})

	Expect(e2eCtx.Environment.ClusterctlConfigPath).To(BeAnExistingFile(), "Failed to create clusterctl config")

	//clusterctl.Init(context.TODO(), clusterctl.InitInput{
	//	LogFolder:            filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),
	//	ClusterctlConfigPath: e2eCtx.Environment.ClusterctlConfigPath,
	//	KubeconfigPath:       e2eCtx.Environment.BootstrapClusterProxy.GetKubeconfigPath(),
	//
	//	CoreProvider:          "cluster-api:v1.11.2",
	//	BootstrapProviders:    []string{"kubeadm:v1.11.2"},
	//	ControlPlaneProviders: []string{"kubeadm:v1.11.2"},
	//
	//	InfrastructureProviders: []string{},
	//})

	clusterctl.InitManagementClusterAndWatchControllerLogs(context.TODO(),

		clusterctl.InitManagementClusterAndWatchControllerLogsInput{

			ClusterProxy: e2eCtx.Environment.BootstrapClusterProxy,

			ClusterctlConfigPath: e2eCtx.Environment.ClusterctlConfigPath,



			InfrastructureProviders: []string{"ironcore-metal:v0.2.0"},

			LogFolder: filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", e2eCtx.Environment.BootstrapClusterProxy.GetName()),

			CoreProvider:            "cluster-api:v1.11.2",
			BootstrapProviders:      []string{"kubeadm:v1.11.2"},
			ControlPlaneProviders:   []string{"kubeadm:v1.11.2"},


		}, e2eCtx.E2EConfig.GetIntervals(e2eCtx.Environment.BootstrapClusterProxy.GetName(), "wait-controllers")...)

	fmt.Printf("\n INSTALLED \n")

})

var _ = AfterSuite(func() {

	if e2eCtx.Environment.BootstrapClusterProxy != nil {

		e2eCtx.Environment.BootstrapClusterProxy.Dispose(context.TODO())

	}

})
