package shared

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// E2EContext represents the context of the e2e test.
type E2EContext struct {
	Settings    Settings
	E2EConfig   *clusterctl.E2EConfig
	Environment RuntimeEnvironment
}

type Settings struct {
	ConfigPath             string
	UseExistingCluster     bool
	ArtifactFolder         string
	DataFolder             string
	SkipCleanup            bool
	GinkgoNodes            int
	GinkgoSlowSpecThreshold int
	KubetestConfigFilePath string
	UseCIArtifacts         bool
	Debug                  bool
}

type RuntimeEnvironment struct {
	BootstrapClusterProvider bootstrap.ClusterProvider
	BootstrapClusterProxy    framework.ClusterProxy
	Namespaces               map[*corev1.Namespace]context.CancelFunc
	ClusterctlConfigPath     string
	Scheme                   *runtime.Scheme
}

// CreateScheme creates a new scheme and adds CAPI and IronCore types to it
func CreateScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1.AddToScheme(scheme))
	return scheme
}