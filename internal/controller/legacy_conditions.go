// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// setMissingConditionReasons sets a reason on every condition that has none and reports whether
// any condition was changed.
//
// Provider versions implementing the v1beta1 contract (v0.2.0 and earlier) wrote CAPI v1beta1
// conditions without a reason, which the metav1.Condition schema requires, so status patches of
// such objects are rejected. The reason is derived from the status. Keeping the conditions instead
// of removing them avoids Ready briefly disappearing during upgrades; the controller overwrites
// them with their real reasons in the next reconcile.
func setMissingConditionReasons(conditions []metav1.Condition) bool {
	changed := false
	for i := range conditions {
		if conditions[i].Reason != "" {
			continue
		}
		switch conditions[i].Status {
		case metav1.ConditionTrue:
			conditions[i].Reason = clusterv1.ReadyReason
		case metav1.ConditionFalse:
			conditions[i].Reason = clusterv1.NotReadyReason
		default:
			conditions[i].Reason = clusterv1.ReadyUnknownReason
		}
		changed = true
	}
	return changed
}
