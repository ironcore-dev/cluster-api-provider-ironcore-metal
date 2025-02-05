// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

// IroncoreMetalClusterStatus defines the observed state of IroncoreMetalCluster
type IroncoreMetalClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the IroncoreMetalCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
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
func (c *IroncoreMetalCluster) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the underlying service state of the IroncoreMetalCluster to the predescribed clusterv1.Conditions.
func (c *IroncoreMetalCluster) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IroncoreMetalCluster{}, &IroncoreMetalClusterList{})
}
