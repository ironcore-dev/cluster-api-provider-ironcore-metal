// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// IroncoreMetalMachineTemplateSpec defines the desired state of IroncoreMetalMachineTemplate
type IroncoreMetalMachineTemplateSpec struct {
	Template IroncoreMetalMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true

// IroncoreMetalMachineTemplate is the Schema for the ironcoremetalmachinetemplates API
type IroncoreMetalMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IroncoreMetalMachineTemplateSpec `json:"spec,omitempty"`
}

// IroncoreMetalMachineTemplateResource defines the spec and metadata for IroncoreMetalMachineTemplate supported by capi.
type IroncoreMetalMachineTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta     `json:"metadata,omitempty"`
	Spec       IroncoreMetalMachineSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// IroncoreMetalMachineTemplateList contains a list of IroncoreMetalMachineTemplate
type IroncoreMetalMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IroncoreMetalMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IroncoreMetalMachineTemplate{}, &IroncoreMetalMachineTemplateList{})
}
