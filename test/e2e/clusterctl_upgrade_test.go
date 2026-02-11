package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
)

var (
	ironcoreReleasev020 string
	capiRelease111      string
)

var _ = Describe("When testing clusterctl upgrades for IroncoreMetal (v0.2.0=>current)", func() {
	//BeforeEach(func(ctx context.Context) {
	//
	//	release, err := clusterctl.ResolveRelease(ctx, "go://github.com/ironcore-dev/cluster-api-provider-ironcore-metal")
	//	Expect(err).ToNot(HaveOccurred(), "failed to get stable release of IroncoreMetal provider")
	//	ironcoreReleasev020 = "v" + release
	//
	//	release, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.11")
	//	Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
	//	capiRelease111 = "v" + release
	//})

	It("should find release", func() {
		fmt.Printf("Iron: %s\n", ironcoreReleasev020)
		fmt.Printf("CAPI: %s\n", capiRelease111)
	})

	//capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
	//	return capi_e2e.ClusterctlUpgradeSpecInput{
	//		E2EConfig:                         e2eCtx.E2EConfig,
	//		ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
	//		BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
	//		ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
	//		SkipCleanup:                       false,
	//		InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease111 + "/clusterctl-{OS}-{ARCH}",
	//		InitWithProvidersContract:         "v1beta1",
	//		InitWithInfrastructureProviders:   []string{"ironcore-metal:v0.2.0"},
	//		InitWithCoreProvider:              "cluster-api:" + capiRelease111,
	//		InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease111},
	//		InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease111},
	//		MgmtFlavor:                        "",
	//		WorkloadFlavor:                    "",
	//		InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable("KUBERNETES_VERSION"),
	//		//InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
	//		UseKindForManagementCluster:       true,
	//	}
	//})

})
