//go:build e2e

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
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
		// The versions come from the providers in the e2e config: CAPI and kubeadm at the version
		// resolved for the current contract, ironcore-metal from its oldest (v1beta1 contract)
		// to its latest (v1beta2 contract, built from source) version.
		currentContract := clusterv1.GroupVersion.Version
		coreProvider := singleProvider(e2eCtx.E2EConfig.GetProviderLatestVersionsByContract(currentContract, "cluster-api"))
		kubeadmProvider := singleProvider(e2eCtx.E2EConfig.GetProviderLatestVersionsByContract(currentContract, "kubeadm"))
		initInfraProvider := singleProvider(e2eCtx.E2EConfig.GetProvidersWithOldestVersion("ironcore-metal"))
		upgradeInfraProvider := singleProvider(e2eCtx.E2EConfig.GetProviderLatestVersionsByContract(currentContract, "ironcore-metal"))

		By(fmt.Sprintf("Initializing the management cluster with %s, %s and %s", coreProvider, kubeadmProvider, initInfraProvider))
		clusterctl.InitManagementClusterAndWatchControllerLogs(ctx,
			clusterctl.InitManagementClusterAndWatchControllerLogsInput{
				ClusterProxy:            managementClusterProxy,
				ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
				CoreProvider:            coreProvider,
				BootstrapProviders:      []string{kubeadmProvider},
				ControlPlaneProviders:   []string{kubeadmProvider},
				InfrastructureProviders: []string{initInfraProvider},
				LogFolder:               managementClusterLogFolder,
			}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-controllers")...,
		)

		By(fmt.Sprintf("Upgrading the infrastructure provider to %s", upgradeInfraProvider))
		clusterctl.UpgradeManagementClusterAndWait(ctx, clusterctl.UpgradeManagementClusterAndWaitInput{
			ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
			ClusterProxy:            managementClusterProxy,
			InfrastructureProviders: []string{upgradeInfraProvider},
			ClusterctlVariables:     e2eCtx.E2EConfig.Variables,
			LogFolder:               managementClusterLogFolder,
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-controllers")...)

		By("\n********** The Management Cluster was successfully upgraded! **********\n\n")
	})
})

// singleProvider returns the only "name:version" entry of a provider lookup from the e2e config.
func singleProvider(providers []string) string {
	ExpectWithOffset(1, providers).To(HaveLen(1), "expected exactly one matching provider version in the e2e config")
	return providers[0]
}
