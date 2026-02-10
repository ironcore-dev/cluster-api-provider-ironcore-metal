// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package scope

import (
	"context"

	"github.com/go-logr/logr"
	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/ironcore-dev/metal-operator/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	//nolint:staticcheck // we use deprecated package intentionally following the CAPI migration strategy
	v1beta1patch "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/patch"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MachineScopeParams defines the input parameters used to create a new Scope.
type MachineScopeParams struct {
	Client               client.Client
	Logger               *logr.Logger
	Cluster              *clusterv1.Cluster
	Machine              *clusterv1.Machine
	IroncoreMetalCluster *infrav1.IroncoreMetalCluster
	IroncoreMetalMachine *infrav1.IroncoreMetalMachine
}

// MachineScope defines the basic context for an actuator to operate upon.
type MachineScope struct {
	*logr.Logger
	client               client.Client
	patchHelper          *v1beta1patch.Helper
	Cluster              *clusterv1.Cluster
	Machine              *clusterv1.Machine
	IroncoreMetalCluster *infrav1.IroncoreMetalCluster
	IroncoreMetalMachine *infrav1.IroncoreMetalMachine
	ServerClaim          *v1alpha1.ServerClaim
}

// NewMachineScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.IroncoreMetalCluster == nil {
		return nil, errors.New("IroncoreMetalCluster is required when creating a MachineScope")
	}
	if params.IroncoreMetalMachine == nil {
		return nil, errors.New("IroncoreMetalMachine is required when creating a MachineScope")
	}
	if params.Logger == nil {
		logger := log.FromContext(context.Background())
		params.Logger = &logger
	}

	machineScope := &MachineScope{
		Logger:               params.Logger,
		client:               params.Client,
		Cluster:              params.Cluster,
		Machine:              params.Machine,
		IroncoreMetalCluster: params.IroncoreMetalCluster,
		IroncoreMetalMachine: params.IroncoreMetalMachine,
	}

	helper, err := v1beta1patch.NewHelper(params.IroncoreMetalMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	machineScope.patchHelper = helper

	return machineScope, nil
}

// SetReady sets the IroncoreMetalMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.IroncoreMetalMachine.Status.Ready = true
}

// SetNotReady sets the IroncoreMetalMachine Ready Status to false.
func (m *MachineScope) SetNotReady() {
	m.IroncoreMetalMachine.Status.Ready = false
}

// SetFailureMessage sets the IroncoreMetalMachine status failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.IroncoreMetalMachine.Status.FailureMessage = ptr.To(v.Error())
}

// SetFailureReason sets the IroncoreMetalMachine status failure reason.
func (m *MachineScope) SetFailureReason(v string) {
	m.IroncoreMetalMachine.Status.FailureReason = v
}

// HasFailed returns the failure state of the machine scope.
func (m *MachineScope) HasFailed() bool {
	return m.IroncoreMetalMachine.Status.FailureReason != "" || m.IroncoreMetalMachine.Status.FailureMessage != nil
}

// PatchObject persists the Machine configuration and status.
func (m *MachineScope) PatchObject() error {
	// always update the readyCondition.
	// TBD readyCondition

	return m.patchHelper.Patch(context.TODO(), m.IroncoreMetalMachine)
}

// Close closes the current scope persisting the Machine configuration and status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}
