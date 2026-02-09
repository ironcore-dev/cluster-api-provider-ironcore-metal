// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1b1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	"sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("IroncoreMetalCluster Controller", func() {

	var (
		ctx                context.Context
		namespace          string
		clusterName        string
		typeNamespacedName types.NamespacedName
		ironcoreCluster    *infrav1.IroncoreMetalCluster
		capiCluster        *clusterv1b1.Cluster
		reconciler         *IroncoreMetalClusterReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		clusterName = "test-cluster-" + uuid.NewString()

		typeNamespacedName = types.NamespacedName{
			Name:      clusterName,
			Namespace: namespace,
		}

		reconciler = &IroncoreMetalClusterReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		capiCluster = &clusterv1b1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}

		ctrlutil.AddFinalizer(capiCluster, "cluster.cluster.x-k8s.io")
		Expect(k8sClient.Create(ctx, capiCluster)).To(Succeed())

		ironcoreCluster = &infrav1.IroncoreMetalCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1b1.GroupVersion.String(),
						Kind:       "Cluster",
						Name:       capiCluster.Name,
						UID:        capiCluster.UID,
					},
				},
			},
		}
	})

	AfterEach(func() {
		if ironcoreCluster != nil {
			err := k8sClient.Delete(ctx, ironcoreCluster)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
		}

		if capiCluster != nil {
			key := client.ObjectKeyFromObject(capiCluster)
			if err := k8sClient.Get(ctx, key, capiCluster); err != nil {
				return
			}

			if ctrlutil.RemoveFinalizer(capiCluster, "cluster.cluster.x-k8s.io") {
				Expect(k8sClient.Update(ctx, capiCluster)).To(Succeed())
			}

			err := k8sClient.Delete(ctx, capiCluster)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
		}
	})

	Context("When reconciling normal", func() {
		It("Should set the Finalizer and mark the status and condition as Ready", func() {
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Finalizer is added")
			Expect(ironcoreCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))

			By("Verifying the Status is Ready")
			Expect(ironcoreCluster.Status.Ready).To(BeTrue())

			By("Verifying the ClusterReady condition")
			condition := conditions.Get(ironcoreCluster, infrav1.IroncoreMetalClusterReady)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(corev1.ConditionTrue))
		})

		It("Should not reconcile if IroncoreMetalCluster has no OwnerReference to Cluster", func() {
			ironcoreCluster.OwnerReferences = []metav1.OwnerReference{}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)).To(Succeed())

			By("Ensuring Finalizer is NOT added")
			Expect(ironcoreCluster.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))

			By("Ensuring Status.Ready is NOT set")
			Expect(ironcoreCluster.Status.Ready).To(BeFalse())

			By("Ensuring the ClusterReady condition is NOT set")
			condition := conditions.Get(ironcoreCluster, infrav1.IroncoreMetalClusterReady)
			Expect(condition).To(BeNil())
		})

		It("Should not reconcile if IroncoreMetalCluster is paused", func() {
			if ironcoreCluster.Annotations == nil {
				ironcoreCluster.Annotations = map[string]string{}
			}
			ironcoreCluster.Annotations[clusterv1b1.PausedAnnotation] = "true"
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)).To(Succeed())

			By("Ensuring Finalizer is NOT added")
			Expect(ironcoreCluster.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))

			By("Ensuring Status.Ready is NOT set")
			Expect(ironcoreCluster.Status.Ready).To(BeFalse())

			By("Ensuring the ClusterReady condition is NOT set")
			condition := conditions.Get(ironcoreCluster, infrav1.IroncoreMetalClusterReady)
			Expect(condition).To(BeNil())
		})
	})

	Context("When reconciling a delete", func() {
		It("should NOT remove finalizer if owning CAPI Cluster is NOT deleted", func() {
			ironcoreCluster.Finalizers = []string{infrav1.ClusterFinalizer}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)).To(Succeed())
			Expect(ironcoreCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
		})

		It("should NOT remove finalizer if child Machines exist", func() {
			ironcoreCluster.Finalizers = []string{infrav1.ClusterFinalizer}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			machine := &infrav1.IroncoreMetalMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-" + clusterName,
					Namespace: namespace,
					Labels:    map[string]string{clusterv1b1.ClusterNameLabel: clusterName},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())

			defer func() {
				_ = k8sClient.Delete(ctx, machine)
			}()

			Expect(k8sClient.Delete(ctx, capiCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Requeue is requested")
			Expect(result.RequeueAfter).To(BeNumerically("==", infrav1.DefaultReconcilerRequeue))

			By("Verifying Finalizer is STILL present")
			Expect(k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)).To(Succeed())
			Expect(ironcoreCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
		})

		It("should remove finalizer if NO child Machines exist", func() {
			ironcoreCluster.Finalizers = []string{infrav1.ClusterFinalizer}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Expect(k8sClient.Delete(ctx, capiCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NO Requeue is requested")
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Verifying Finalizer is REMOVED")
			err = k8sClient.Get(ctx, typeNamespacedName, ironcoreCluster)
			if !apierrors.IsNotFound(err) {
				Expect(ironcoreCluster.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))
			}
		})
	})
})
