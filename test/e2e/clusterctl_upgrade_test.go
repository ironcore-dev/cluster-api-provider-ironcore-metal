//go:build e2e

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

var _ = Describe("Management Cluster Upgrade Test", func() {
	var (
		ctx                        context.Context
		specName                   string
		managementClusterName      string
		managementClusterProvider  bootstrap.ClusterProvider
		managementClusterProxy     framework.ClusterProxy
		managementClusterLogFolder string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		specName = "clusterctl-upgrade"
		managementClusterName = "clusterctl-upgrade"
		managementClusterLogFolder = filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", managementClusterName)

		By("Creating a dedicated kind cluster for the upgrade test")
		managementClusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(ctx,
			bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
				Name:              managementClusterName,
				KubernetesVersion: e2eCtx.E2EConfig.MustGetVariable("KUBERNETES_VERSION_MANAGEMENT"),
				Images:            e2eCtx.E2EConfig.Images,
				LogFolder:         filepath.Join(managementClusterLogFolder, "kind-logs"),
			})
		Expect(managementClusterProvider).ToNot(BeNil(), "Failed to create a kind cluster")

		By("Creating a management cluster proxy")
		managementClusterProxy = framework.NewClusterProxy(managementClusterName, managementClusterProvider.GetKubeconfigPath(), e2eCtx.Environment.Scheme)
		Expect(managementClusterProxy).ToNot(BeNil(), "Failed to get a kind cluster proxy")
	})

	AfterEach(func() {
		if e2eCtx.Settings.SkipCleanup {
			fmt.Printf("\n********** Skipping cleanup as requested by flag **********\n\n")
			return
		}

		if managementClusterProxy != nil {
			managementClusterProxy.Dispose(ctx)
		}
		if managementClusterProvider != nil {
			managementClusterProvider.Dispose(ctx)
		}

		fmt.Printf("\n********** All resources are cleaned up **********\n\n")
	})

	It("Should successfully upgrade the provider", func() {
		capiVersion := e2eCtx.E2EConfig.MustGetVariable("CAPI_VERSION")
		kubeadmVersion := e2eCtx.E2EConfig.MustGetVariable("KUBEADM_VERSION")
		initVersion := e2eCtx.E2EConfig.MustGetVariable("OLD_PROVIDER_VERSION")
		upgradeVersion := e2eCtx.E2EConfig.MustGetVariable("NEW_PROVIDER_VERSION")

		By(fmt.Sprintf("Initializing the management cluster with older provider version (%s)", initVersion))
		clusterctl.InitManagementClusterAndWatchControllerLogs(ctx,
			clusterctl.InitManagementClusterAndWatchControllerLogsInput{
				ClusterProxy:            managementClusterProxy,
				ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
				CoreProvider:            "cluster-api:" + capiVersion,
				BootstrapProviders:      []string{"kubeadm:" + kubeadmVersion},
				ControlPlaneProviders:   []string{"kubeadm:" + kubeadmVersion},
				InfrastructureProviders: []string{"ironcore-metal:" + initVersion},
				LogFolder:               managementClusterLogFolder,
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-controllers")...,
		)

		By(fmt.Sprintf("Upgrading the providers to the new version (%s)", upgradeVersion))
		clusterctl.UpgradeManagementClusterAndWait(ctx, clusterctl.UpgradeManagementClusterAndWaitInput{
			ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
			ClusterProxy:            managementClusterProxy,
			InfrastructureProviders: []string{"ironcore-metal:" + upgradeVersion},
			ClusterctlVariables:     e2eCtx.E2EConfig.Variables,
			LogFolder:               managementClusterLogFolder,
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-controllers")...)

		By("\n********** The Management Cluster was successfully upgraded! **********\n\n")
	})
})
