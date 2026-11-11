// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// conditionsTestNamespace is the namespace of the objects held by the fake client.
const conditionsTestNamespace = "default"

var _ = Describe("setMissingConditionReasons", func() {
	condition := func(conditionType string, status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Type: conditionType, Status: status, Reason: reason}
	}

	DescribeTable("sets reasons only where they are missing",
		func(conditions []metav1.Condition, expected []metav1.Condition, expectedChanged bool) {
			Expect(setMissingConditionReasons(conditions)).To(Equal(expectedChanged))
			Expect(conditions).To(Equal(expected))
		},
		Entry("no conditions",
			nil, nil, false),
		Entry("all conditions have a reason",
			[]metav1.Condition{condition("Ready", metav1.ConditionTrue, "Custom")},
			[]metav1.Condition{condition("Ready", metav1.ConditionTrue, "Custom")},
			false),
		Entry("True gets Ready",
			[]metav1.Condition{condition("Ready", metav1.ConditionTrue, "")},
			[]metav1.Condition{condition("Ready", metav1.ConditionTrue, clusterv1.ReadyReason)},
			true),
		Entry("False gets NotReady",
			[]metav1.Condition{condition("Ready", metav1.ConditionFalse, "")},
			[]metav1.Condition{condition("Ready", metav1.ConditionFalse, clusterv1.NotReadyReason)},
			true),
		Entry("Unknown gets ReadyUnknown",
			[]metav1.Condition{condition("Ready", metav1.ConditionUnknown, "")},
			[]metav1.Condition{condition("Ready", metav1.ConditionUnknown, clusterv1.ReadyUnknownReason)},
			true),
		Entry("only conditions without a reason are changed",
			[]metav1.Condition{condition("ClusterReady", metav1.ConditionTrue, ""), condition("Paused", metav1.ConditionFalse, clusterv1.NotPausedReason)},
			[]metav1.Condition{condition("ClusterReady", metav1.ConditionTrue, clusterv1.ReadyReason), condition("Paused", metav1.ConditionFalse, clusterv1.NotPausedReason)},
			true),
	)
})

var _ = Describe("IroncoreMetalClusterReconciler.ensureConditionReasons", func() {
	// The CRD rejects conditions without a reason, so this uses a fake client to hold an
	// IroncoreMetalCluster as left behind by v0.2.0.
	It("should persist the missing reasons", func() {
		metalCluster := &infrav1.IroncoreMetalCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "v1beta1-cluster", Namespace: conditionsTestNamespace},
			Status: infrav1.IroncoreMetalClusterStatus{
				Conditions: []metav1.Condition{
					{Type: infrav1.IroncoreMetalClusterReady, Status: metav1.ConditionTrue, LastTransitionTime: metav1.Now()},
					{Type: clusterv1.ReadyCondition, Status: metav1.ConditionTrue, LastTransitionTime: metav1.Now()},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(k8sClient.Scheme()).
			WithObjects(metalCluster).
			WithStatusSubresource(metalCluster).
			Build()
		reconciler := &IroncoreMetalClusterReconciler{Client: fakeClient, Scheme: k8sClient.Scheme()}

		Expect(reconciler.ensureConditionReasons(ctx, metalCluster)).To(Succeed())

		persisted := &infrav1.IroncoreMetalCluster{}
		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(metalCluster), persisted)).To(Succeed())
		Expect(persisted.Status.Conditions).To(ConsistOf(
			HaveField("Reason", clusterv1.ReadyReason),
			HaveField("Reason", clusterv1.ReadyReason),
		))
	})

	It("should not patch when all conditions have a reason", func() {
		metalCluster := &infrav1.IroncoreMetalCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "v1beta2-cluster", Namespace: conditionsTestNamespace},
			Status: infrav1.IroncoreMetalClusterStatus{
				Conditions: []metav1.Condition{
					{Type: clusterv1.ReadyCondition, Status: metav1.ConditionTrue, Reason: clusterv1.ReadyReason, LastTransitionTime: metav1.Now()},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(k8sClient.Scheme()).
			WithObjects(metalCluster).
			WithStatusSubresource(metalCluster).
			Build()
		reconciler := &IroncoreMetalClusterReconciler{Client: fakeClient, Scheme: k8sClient.Scheme()}
		resourceVersion := metalCluster.ResourceVersion

		Expect(reconciler.ensureConditionReasons(ctx, metalCluster)).To(Succeed())
		Expect(metalCluster.ResourceVersion).To(Equal(resourceVersion))
	})
})
