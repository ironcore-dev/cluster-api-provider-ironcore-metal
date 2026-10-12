// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testHost             = "1.2.3.4"
	testServiceDomain    = "test.domain"
	capiClusterFinalizer = "cluster.cluster.x-k8s.io"
)

var _ = Describe("IroncoreMetalCluster Controller", func() {

	var (
		namespace          string
		clusterName        string
		typeNamespacedName types.NamespacedName
		ironcoreCluster    *infrav1.IroncoreMetalCluster
		capiCluster        *clusterv1.Cluster
	)

	// getCluster fetches the current IroncoreMetalCluster for use inside Eventually/Consistently.
	getCluster := func(g Gomega) *infrav1.IroncoreMetalCluster {
		obj := &infrav1.IroncoreMetalCluster{}
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, obj)).To(Succeed())
		return obj
	}

	BeforeEach(func() {
		namespace = "default"
		clusterName = "test-cluster-" + uuid.NewString()

		typeNamespacedName = types.NamespacedName{
			Name:      clusterName,
			Namespace: namespace,
		}

		capiCluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
			Spec: clusterv1.ClusterSpec{
				ControlPlaneRef: clusterv1.ContractVersionedObjectReference{
					APIGroup: infrav1.GroupVersion.Group,
					Kind:     "KubeadmControlPlane",
					Name:     clusterName + "-cp",
				},
			},
		}
		ctrlutil.AddFinalizer(capiCluster, capiClusterFinalizer)
		Expect(k8sClient.Create(ctx, capiCluster)).To(Succeed())

		ironcoreCluster = &infrav1.IroncoreMetalCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Cluster",
						Name:       capiCluster.Name,
						UID:        capiCluster.UID,
					},
				},
			},
			Spec: infrav1.IroncoreMetalClusterSpec{
				ControlPlaneEndpoint: clusterv1.APIEndpoint{
					Host: testHost,
				},
				ClusterNetwork: clusterv1.ClusterNetwork{
					ServiceDomain: testServiceDomain,
				},
			},
		}
	})

	AfterEach(func() {
		// Mirror real CAPI ordering: the Cluster gets a deletionTimestamp but stays
		// around (its finalizer is held) until the InfraCluster is gone.
		if capiCluster != nil {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, capiCluster))).To(Succeed())
		}

		if ironcoreCluster != nil {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, ironcoreCluster))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, &infrav1.IroncoreMetalCluster{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue(), "IroncoreMetalCluster should be cleaned up")
		}

		if capiCluster != nil {
			current := &clusterv1.Cluster{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(capiCluster), current); err == nil {
				if ctrlutil.RemoveFinalizer(current, capiClusterFinalizer) {
					Expect(k8sClient.Update(ctx, current)).To(Succeed())
				}
			}
		}
	})

	Context("When reconciling normal", func() {
		It("Should set the Finalizer and mark the status and conditions as Ready", func() {
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				obj := getCluster(g)

				By("Verifying the Finalizer is added")
				g.Expect(obj.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))

				By("Verifying the deprecated Status.Ready is set")
				g.Expect(obj.Status.Ready).To(BeTrue())

				By("Verifying the v1beta2 initialization field is set")
				g.Expect(obj.Status.Initialization.Provisioned).To(HaveValue(BeTrue()))

				By("Verifying the ClusterReady condition")
				condition := conditions.Get(obj, infrav1.IroncoreMetalClusterReady)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(infrav1.IroncoreMetalClusterReadyReason))

				By("Verifying the summarized Ready condition")
				condition = conditions.Get(obj, clusterv1.ReadyCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))

				By("Verifying the Paused condition is False")
				condition = conditions.Get(obj, clusterv1.PausedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(clusterv1.NotPausedReason))

				By("Verifying the Deleting condition is not set")
				g.Expect(conditions.Get(obj, clusterv1.DeletingCondition)).To(BeNil())

				By("Verifying observedGeneration matches the object generation")
				g.Expect(obj.Status.ObservedGeneration).To(Equal(obj.Generation))
			}).Should(Succeed())
		})

		It("Should not reconcile if IroncoreMetalCluster has no OwnerReference to Cluster", func() {
			ironcoreCluster.OwnerReferences = []metav1.OwnerReference{}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Consistently(func(g Gomega) {
				obj := getCluster(g)
				g.Expect(obj.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))
				g.Expect(obj.Status.Ready).To(BeFalse())
				g.Expect(obj.Status.Initialization.Provisioned).To(BeNil())
				g.Expect(obj.Status.Conditions).To(BeEmpty())
			}).Should(Succeed())
		})
	})

	Context("When the object or its owning Cluster is paused", func() {
		It("Should only set the Paused condition if IroncoreMetalCluster is paused", func() {
			ironcoreCluster.Annotations = map[string]string{clusterv1.PausedAnnotation: "true"}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				condition := conditions.Get(getCluster(g), clusterv1.PausedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(clusterv1.PausedReason))
			}).Should(Succeed())

			By("Ensuring nothing else is reconciled while paused")
			Consistently(func(g Gomega) {
				obj := getCluster(g)
				g.Expect(obj.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))
				g.Expect(obj.Status.Ready).To(BeFalse())
				g.Expect(obj.Status.Initialization.Provisioned).To(BeNil())
				g.Expect(conditions.Get(obj, infrav1.IroncoreMetalClusterReady)).To(BeNil())
				g.Expect(conditions.Get(obj, clusterv1.ReadyCondition)).To(BeNil())
			}).Should(Succeed())
		})

		It("Should only set the Paused condition if the owning Cluster is paused", func() {
			capiCluster.Spec.Paused = ptr.To(true)
			Expect(k8sClient.Update(ctx, capiCluster)).To(Succeed())
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				condition := conditions.Get(getCluster(g), clusterv1.PausedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(clusterv1.PausedReason))
			}).Should(Succeed())

			Consistently(func(g Gomega) {
				obj := getCluster(g)
				g.Expect(obj.Finalizers).NotTo(ContainElement(infrav1.ClusterFinalizer))
				g.Expect(obj.Status.Ready).To(BeFalse())
				g.Expect(conditions.Get(obj, clusterv1.ReadyCondition)).To(BeNil())
			}).Should(Succeed())
		})

		It("Should flip the Paused condition to False and reconcile after unpausing", func() {
			ironcoreCluster.Annotations = map[string]string{clusterv1.PausedAnnotation: "true"}
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				condition := conditions.Get(getCluster(g), clusterv1.PausedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			}).Should(Succeed())

			By("Removing the paused annotation")
			Eventually(func(g Gomega) {
				obj := getCluster(g)
				delete(obj.Annotations, clusterv1.PausedAnnotation)
				g.Expect(k8sClient.Update(ctx, obj)).To(Succeed())
			}).Should(Succeed())

			Eventually(func(g Gomega) {
				obj := getCluster(g)

				By("Ensuring the Paused condition is False")
				condition := conditions.Get(obj, clusterv1.PausedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(clusterv1.NotPausedReason))

				By("Ensuring the object was reconciled normally")
				g.Expect(obj.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
				g.Expect(obj.Status.Ready).To(BeTrue())
				g.Expect(obj.Status.Initialization.Provisioned).To(HaveValue(BeTrue()))
				g.Expect(conditions.Get(obj, clusterv1.ReadyCondition).Status).To(Equal(metav1.ConditionTrue))
			}).Should(Succeed())
		})
	})

	Context("When reconciling a delete", func() {
		// waitForReady lets the controller finish the normal reconcile (finalizer added)
		// before the test triggers deletion, so deletion goes through the real flow.
		waitForReady := func() {
			Eventually(func(g Gomega) {
				obj := getCluster(g)
				g.Expect(obj.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
				g.Expect(conditions.Get(obj, clusterv1.ReadyCondition).Status).To(Equal(metav1.ConditionTrue))
			}).Should(Succeed())
		}

		It("should NOT remove finalizer if owning CAPI Cluster is NOT deleted", func() {
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())
			waitForReady()

			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				obj := getCluster(g)

				By("Verifying the Deleting condition")
				condition := conditions.Get(obj, clusterv1.DeletingCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(infrav1.WaitingForOwnerClusterDeletionReason))

				By("Verifying the summarized Ready condition is False")
				condition = conditions.Get(obj, clusterv1.ReadyCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			}).Should(Succeed())

			By("Verifying Finalizer is STILL present")
			Consistently(func(g Gomega) {
				g.Expect(getCluster(g).Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
			}).Should(Succeed())
		})

		It("should NOT remove finalizer if child Machines exist", func() {
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())
			waitForReady()

			machine := &infrav1.IroncoreMetalMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-" + clusterName,
					Namespace: namespace,
					Labels:    map[string]string{clusterv1.ClusterNameLabel: clusterName},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			DeferCleanup(func() {
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, machine))).To(Succeed())
			})

			Expect(k8sClient.Delete(ctx, capiCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			Eventually(func(g Gomega) {
				obj := getCluster(g)

				By("Verifying the Deleting condition reports the waiting machines")
				condition := conditions.Get(obj, clusterv1.DeletingCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(infrav1.WaitingForMachinesDeletionReason))
				g.Expect(condition.Message).To(ContainSubstring("1 IroncoreMetalMachine"))

				By("Verifying the summarized Ready condition is False")
				condition = conditions.Get(obj, clusterv1.ReadyCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			}).Should(Succeed())

			By("Verifying Finalizer is STILL present while the machine exists")
			Consistently(func(g Gomega) {
				g.Expect(getCluster(g).Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
			}).Should(Succeed())

			By("Verifying the finalizer is removed once the machine is gone")
			Expect(k8sClient.Delete(ctx, machine)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, &infrav1.IroncoreMetalCluster{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})

		It("should remove finalizer if NO child Machines exist", func() {
			Expect(k8sClient.Create(ctx, ironcoreCluster)).To(Succeed())
			waitForReady()

			Expect(k8sClient.Delete(ctx, capiCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ironcoreCluster)).To(Succeed())

			By("Verifying the object is deleted once the finalizer is removed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, &infrav1.IroncoreMetalCluster{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})
})
