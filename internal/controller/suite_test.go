// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	infrav1alpha1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/ironcore-dev/controller-utils/modutils"
	metalv1alpha1 "github.com/ironcore-dev/metal-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clusterapiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

const (
	eventuallyTimeout    = 20 * time.Second
	pollingInterval      = 250 * time.Millisecond
	consistentlyDuration = 1 * time.Second
)

var (
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	ctrl.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			modutils.Dir("sigs.k8s.io/cluster-api", "config", "crd", "bases"),
			modutils.Dir("github.com/ironcore-dev/metal-operator", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.32.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// +kubebuilder:scaffold:scheme
	Expect(clusterapiv1beta2.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(infrav1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(metalv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(capiv1beta2.AddToScheme(scheme.Scheme)).To(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// set komega client
	SetClient(k8sClient)

	By("starting the manager with the IroncoreMetalCluster controller")
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // disable the metrics endpoint in tests
		},
	})
	Expect(err).NotTo(HaveOccurred())

	Expect((&IroncoreMetalClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(ctx, mgr)).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
	Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// Stop the manager before the API server goes away.
	cancel()
	Expect(testEnv.Stop()).To(Succeed())
})
