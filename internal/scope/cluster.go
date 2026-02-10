// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package scope

import (
	"context"

	"github.com/go-logr/logr"
	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	//nolint:staticcheck // we use deprecated package intentionally following the CAPI migration strategy
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	//nolint:staticcheck // we use deprecated package intentionally following the CAPI migration strategy
	v1beta1patch "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/patch"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client               client.Client
	Logger               *logr.Logger
	Cluster              *clusterv1.Cluster
	IroncoreMetalCluster *infrav1.IroncoreMetalCluster
	ControllerName       string
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	*logr.Logger
	client               client.Client
	patchHelper          *v1beta1patch.Helper
	Cluster              *clusterv1.Cluster
	IroncoreMetalCluster *infrav1.IroncoreMetalCluster
	controllerName       string
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a ClusterScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}
	if params.IroncoreMetalCluster == nil {
		return nil, errors.New("IroncoreMetalCluster is required when creating a ClusterScope")
	}
	if params.Logger == nil {
		logger := log.FromContext(context.Background())
		params.Logger = &logger
	}

	clusterScope := &ClusterScope{
		Logger:               params.Logger,
		client:               params.Client,
		Cluster:              params.Cluster,
		IroncoreMetalCluster: params.IroncoreMetalCluster,
		controllerName:       params.ControllerName,
	}

	helper, err := v1beta1patch.NewHelper(params.IroncoreMetalCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	clusterScope.patchHelper = helper

	return clusterScope, nil
}

// Name returns the CAPI cluster name.
func (s *ClusterScope) Name() string {
	return s.Cluster.Name
}

// Namespace returns the cluster namespace.
func (s *ClusterScope) Namespace() string {
	return s.Cluster.Namespace
}

// InfraClusterName returns the name of the Metal cluster.
func (s *ClusterScope) InfraClusterName() string {
	return s.IroncoreMetalCluster.Name
}

// KubernetesClusterName is the name of the Kubernetes cluster. For the cluster
// scope this is the same as the CAPI cluster name.
func (s *ClusterScope) KubernetesClusterName() string {
	return s.Cluster.Name
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	// always update the readyCondition.
	v1beta1conditions.SetSummary(s.IroncoreMetalCluster,
		v1beta1conditions.WithConditions(
			infrav1.IroncoreMetalClusterReady,
		),
	)

	return s.patchHelper.Patch(context.TODO(), s.IroncoreMetalCluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}
