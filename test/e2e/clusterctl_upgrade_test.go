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

	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// upgradeTestObjectsName is the name of the namespace, Cluster and IroncoreMetalCluster
	// created before the upgrade to check how existing objects are migrated.
	upgradeTestObjectsName = "upgrade-test"
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

	It("Should successfully upgrade the provider and its existing objects", func() {
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

		mgmtClient := managementClusterProxy.GetClient()
		cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: upgradeTestObjectsName, Namespace: upgradeTestObjectsName}}
		metalCluster := &infrav1.IroncoreMetalCluster{ObjectMeta: metav1.ObjectMeta{Name: upgradeTestObjectsName, Namespace: upgradeTestObjectsName}}

		By("Creating a Cluster with an IroncoreMetalCluster on the old provider version")
		createClusterWithIroncoreMetalCluster(ctx, mgmtClient, upgradeTestObjectsName)

		By("Waiting for the Cluster infrastructure to be provisioned through the v1beta1 contract")
		Eventually(func(g Gomega) {
			g.Expect(mgmtClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)).To(Succeed())
			g.Expect(ptr.Deref(cluster.Status.Initialization.InfrastructureProvisioned, false)).To(BeTrue())
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-infrastructure-provisioned")...).Should(Succeed())

		By(fmt.Sprintf("Upgrading the infrastructure provider to %s", upgradeInfraProvider))
		clusterctl.UpgradeManagementClusterAndWait(ctx, clusterctl.UpgradeManagementClusterAndWaitInput{
			ClusterctlConfigPath:    e2eCtx.Environment.ClusterctlConfigPath,
			ClusterProxy:            managementClusterProxy,
			InfrastructureProviders: []string{upgradeInfraProvider},
			ClusterctlVariables:     e2eCtx.E2EConfig.Variables,
			LogFolder:               managementClusterLogFolder,
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-controllers")...)

		By("Waiting for the existing IroncoreMetalCluster to be reconciled through the v1beta2 contract")
		Eventually(func(g Gomega) {
			g.Expect(mgmtClient.Get(ctx, client.ObjectKeyFromObject(metalCluster), metalCluster)).To(Succeed())
			g.Expect(ptr.Deref(metalCluster.Status.Initialization.Provisioned, false)).To(BeTrue())
			g.Expect(metalCluster.Status.ObservedGeneration).To(Equal(metalCluster.Generation))

			// The v1beta1 conditions written by the old version have no reason; the new
			// version must have replaced them with valid v1beta2 conditions.
			expectCondition(g, metalCluster, clusterv1.ReadyCondition, metav1.ConditionTrue)
			expectCondition(g, metalCluster, clusterv1.PausedCondition, metav1.ConditionFalse)
			expectCondition(g, metalCluster, infrav1.IroncoreMetalClusterReady, metav1.ConditionTrue)
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-infrastructure-provisioned")...).Should(Succeed())

		By("Checking that the Cluster still reports its infrastructure as ready")
		Eventually(func(g Gomega) {
			g.Expect(mgmtClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)).To(Succeed())
			g.Expect(ptr.Deref(cluster.Status.Initialization.InfrastructureProvisioned, false)).To(BeTrue())
			expectCondition(g, cluster, clusterv1.InfrastructureReadyCondition, metav1.ConditionTrue)
		}, e2eCtx.E2EConfig.GetIntervals(specName, "wait-infrastructure-provisioned")...).Should(Succeed())

		By("\n********** The Management Cluster was successfully upgraded! **********\n\n")
	})
})

// singleProvider returns the only "name:version" entry of a provider lookup from the e2e config.
func singleProvider(providers []string) string {
	ExpectWithOffset(1, providers).To(HaveLen(1), "expected exactly one matching provider version in the e2e config")
	return providers[0]
}

// createClusterWithIroncoreMetalCluster creates a namespace with a Cluster that only references an
// IroncoreMetalCluster. It needs no servers: the IroncoreMetalCluster becomes ready on its own.
// The IroncoreMetalCluster is created unstructured, so it is valid for the CRD of any provider version.
func createClusterWithIroncoreMetalCluster(ctx context.Context, c client.Client, name string) {
	ExpectWithOffset(1, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})).To(Succeed())

	metalCluster := &unstructured.Unstructured{}
	metalCluster.SetGroupVersionKind(infrav1.GroupVersion.WithKind("IroncoreMetalCluster"))
	metalCluster.SetName(name)
	metalCluster.SetNamespace(name)
	ExpectWithOffset(1, unstructured.SetNestedMap(metalCluster.Object, map[string]any{
		"host": "foo",
		"port": int64(443),
	}, "spec", "controlPlaneEndpoint")).To(Succeed())
	ExpectWithOffset(1, c.Create(ctx, metalCluster)).To(Succeed())

	ExpectWithOffset(1, c.Create(ctx, &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{
				APIGroup: infrav1.GroupVersion.Group,
				Kind:     "IroncoreMetalCluster",
				Name:     name,
			},
		},
	})).To(Succeed())
}

// expectCondition asserts that obj has a condition of the given type and status with a non-empty reason.
func expectCondition(g Gomega, obj conditions.Getter, conditionType string, status metav1.ConditionStatus) {
	condition := conditions.Get(obj, conditionType)
	g.ExpectWithOffset(1, condition).NotTo(BeNil(), "condition %s not set", conditionType)
	g.ExpectWithOffset(1, condition.Status).To(Equal(status), "condition %s", conditionType)
	g.ExpectWithOffset(1, condition.Reason).NotTo(BeEmpty(), "condition %s has no reason", conditionType)
}
