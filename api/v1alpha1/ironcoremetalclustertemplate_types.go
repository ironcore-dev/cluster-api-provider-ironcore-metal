// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// IroncoreMetalClusterTemplateResource describes the data needed to create a IroncoreMetalCluster from a template.
type IroncoreMetalClusterTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta     `json:"metadata,omitempty,omitzero"`
	Spec       IroncoreMetalClusterSpec `json:"spec"`
}

// IroncoreMetalClusterTemplateSpec defines the desired state of IroncoreMetalClusterTemplate.
type IroncoreMetalClusterTemplateSpec struct {
	Template IroncoreMetalClusterTemplateResource `json:"template"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:path=ironcoremetalclustertemplates,scope=Namespaced,categories=cluster-api,shortName=imct
// IroncoreMetalClusterTemplate is the Schema for the ironcoremetalclustertemplates API.
type IroncoreMetalClusterTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// metadata is the standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec is the desired state of the IroncoreMetalClusterTemplate.
	// +optional
	Spec IroncoreMetalClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// IroncoreMetalClusterTemplateList contains a list of IroncoreMetalClusterTemplate.
type IroncoreMetalClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +required
	Items []IroncoreMetalClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &IroncoreMetalClusterTemplate{}, &IroncoreMetalClusterTemplateList{})
		return nil
	})
}
