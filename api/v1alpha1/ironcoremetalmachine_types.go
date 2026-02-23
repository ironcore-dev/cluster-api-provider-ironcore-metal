// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MachineFinalizer allows ReconcileIroncoreMetalMachine to clean up resources associated with IroncoreMetalMachine before
	// removing it from the apiserver.
	MachineFinalizer = "ironcoremetalmachine.infrastructure.cluster.x-k8s.io"

	// DefaultReconcilerRequeue is the default value for the reconcile retry.
	DefaultReconcilerRequeue = 5 * time.Second
)

// IroncoreMetalMachineSpec defines the desired state of IroncoreMetalMachine
type IroncoreMetalMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID string `json:"providerID,omitempty"`

	// Image specifies the boot image to be used for the server.
	Image string `json:"image"`

	// ServerSelector specifies matching criteria for labels on Servers.
	// This is used to claim specific Server types for a IroncoreMetalMachine.
	// +optional
	ServerSelector *metav1.LabelSelector `json:"serverSelector,omitempty"`

	// IPAMConfig is a list of references to Network resources that should be used to assign IP addresses to the worker nodes.
	// +optional
	IPAMConfig []IPAMConfig `json:"ipamConfig,omitempty"`
	// Metadata is a key-value map of additional data which should be passed to the Machine.
	// +optional
	Metadata *apiextensionsv1.JSON `json:"metadata,omitempty"`
}

// IroncoreMetalMachineInitializationStatus provides observations of the IroncoreMetalMachine initialization process.
type IroncoreMetalMachineInitializationStatus struct {
	// Provisioned is true when the infrastructure provider reports that the Machine's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// IroncoreMetalMachineStatus defines the observed state of IroncoreMetalMachine
type IroncoreMetalMachineStatus struct {
	// Ready indicates the Machine infrastructure has been provisioned and is ready.
	// Deprecated: This field is part of the v1beta1 contract and will be removed in the future.
	// +optional
	Ready bool `json:"ready"`

	// Initialization provides observations of the IroncoreMetalMachine initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Machine provisioning.
	// +optional
	Initialization IroncoreMetalMachineInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Conditions defines current service state of the IroncoreMetalMachine
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IroncoreMetalMachine is the Schema for the ironcoremetalmachines API
type IroncoreMetalMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IroncoreMetalMachineSpec   `json:"spec,omitempty"`
	Status IroncoreMetalMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IroncoreMetalMachineList contains a list of IroncoreMetalMachine
type IroncoreMetalMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IroncoreMetalMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IroncoreMetalMachine{}, &IroncoreMetalMachineList{})
}

// IPAMObjectReference is a reference to the IPAM object, which will be used for IP allocation.
type IPAMObjectReference struct {
	// Name is the name of resource being referenced.
	Name string `json:"name"`
	// APIGroup is the group for the resource being referenced.
	APIGroup string `json:"apiGroup"`
	// Kind is the type of resource being referenced.
	Kind string `json:"kind"`
}

// IPAMConfig is a reference to an IPAM resource.
type IPAMConfig struct {
	// MetadataKey is the name of metadata key for the network.
	MetadataKey string `json:"metadataKey"`
	// IPAMRef is a reference to the IPAM object, which will be used for IP allocation.
	IPAMRef *IPAMObjectReference `json:"ipamRef"`
}
