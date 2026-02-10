// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//nolint:staticcheck // we use deprecated package intentionally following the CAPI migration strategy
	clusterv1b1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1b2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// ClusterFinalizer allows IroncoreMetalClusterReconciler to clean up resources associated with IroncoreMetalCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "ironcoremetalcluster.infrastructure.cluster.x-k8s.io"
)

// IroncoreMetalClusterSpec defines the desired state of IroncoreMetalCluster
type IroncoreMetalClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1b2.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
	// Cluster network configuration.
	// +optional
	ClusterNetwork clusterv1b2.ClusterNetwork `json:"clusterNetwork,omitempty"`
}

// IroncoreMetalClusterStatus defines the observed state of IroncoreMetalCluster
type IroncoreMetalClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the IroncoreMetalCluster.
	// This field is kept for backward compatibility with CAPI v1beta1.
	// +optional
	Conditions clusterv1b1.Conditions `json:"conditions,omitempty"`

	// V1Beta2 contains the status fields for the CAPI v1beta2 API.
	// This field is added as part of the migration to CAPI v1.11.
	// It stores conditions in the new metav1.Condition format.
	// +optional
	V1Beta2 *IroncoreMetalClusterV1Beta2Status `json:"v1beta2,omitempty"`
}

// IroncoreMetalClusterV1Beta2Status holds the status fields specific to the CAPI v1beta2
type IroncoreMetalClusterV1Beta2Status struct {
	// Conditions stores the conditions in the new metav1.Condition format.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IroncoreMetalCluster is the Schema for the ironcoremetalclusters API
type IroncoreMetalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IroncoreMetalClusterSpec   `json:"spec,omitempty"`
	Status IroncoreMetalClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IroncoreMetalClusterList contains a list of IroncoreMetalCluster
type IroncoreMetalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IroncoreMetalCluster `json:"items"`
}

// GetConditions returns the observations of the operational state of the IroncoreMetalCluster resource.
func (c *IroncoreMetalCluster) GetConditions() clusterv1b1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the underlying service state of the IroncoreMetalCluster to the predescribed clusterv1b1.Conditions.
func (c *IroncoreMetalCluster) SetConditions(conditions clusterv1b1.Conditions) {
	c.Status.Conditions = conditions
}

// GetV1Beta2Conditions returns the conditions in the new v1beta2 format.
// This satisfies the Getter interface for the v1beta2 conditions utility.
func (c *IroncoreMetalCluster) GetV1Beta2Conditions() []metav1.Condition {
	if c.Status.V1Beta2 == nil {
		return nil
	}
	return c.Status.V1Beta2.Conditions
}

// SetV1Beta2Conditions sets the conditions in the new v1beta2 format.
// This satisfies the Setter interface for the v1beta2 conditions utility.
func (c *IroncoreMetalCluster) SetV1Beta2Conditions(conditions []metav1.Condition) {
	if c.Status.V1Beta2 == nil {
		c.Status.V1Beta2 = &IroncoreMetalClusterV1Beta2Status{}
	}
	c.Status.V1Beta2.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IroncoreMetalCluster{}, &IroncoreMetalClusterList{})
}
