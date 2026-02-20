// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
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
	// Cluster network configuration.
	// +optional
	ClusterNetwork clusterv1.ClusterNetwork `json:"clusterNetwork,omitempty"`
}

// IroncoreMetalClusterInitializationStatus provides observations of the IroncoreMetalCluster initialization process.
type IroncoreMetalClusterInitializationStatus struct {
	// Provisioned is true when the infrastructure provider reports that the Cluster's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Cluster provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// IroncoreMetalClusterStatus defines the observed state of IroncoreMetalCluster
type IroncoreMetalClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready.
	// Deprecated: This field is part of the v1beta1 contract and will be ignored in the future.
	// +optional
	Ready bool `json:"ready"`

	// Initialization provides observations of the IroncoreMetalCluster initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Cluster provisioning.
	// +optional
	Initialization IroncoreMetalClusterInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Conditions defines current service state of the IroncoreMetalCluster.
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
func (c *IroncoreMetalCluster) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

// SetConditions sets the underlying service state of the IroncoreMetalCluster to the predescribed clusterv1b1.Conditions.
func (c *IroncoreMetalCluster) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IroncoreMetalCluster{}, &IroncoreMetalClusterList{})
}
