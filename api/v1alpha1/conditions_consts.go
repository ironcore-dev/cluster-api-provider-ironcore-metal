// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

// IroncoreMetalCluster conditions.

const (
	// IroncoreMetalClusterReady documents the status of IroncoreMetalCluster and its underlying resources.
	IroncoreMetalClusterReady string = "ClusterReady"

	IroncoreMetalClusterReadyReason    = "Ready"
	IroncoreMetalClusterDeletingReason = "Deleting"

	// Reasons for the Deleting condition.
	WaitingForMachinesDeletionReason     = "WaitingForMachinesDeletion"
	WaitingForOwnerClusterDeletionReason = "WaitingForOwnerClusterDeletion"
)

// IroncoreMetalMachine conditions.

// IroncoreMetalMachine's Ready condition and corresponding reasons.
const (
	// IroncoreMetalMachineReadyCondition is true if the IroncoreMetalMachine's IPAddressesReady and
	// ServerClaimReady conditions are true.
	IroncoreMetalMachineReadyCondition = clusterv1.ReadyCondition

	// IroncoreMetalMachineReadyReason surfaces when the IroncoreMetalMachine readiness criteria are met.
	IroncoreMetalMachineReadyReason = clusterv1.ReadyReason

	// IroncoreMetalMachineNotReadyReason surfaces when the IroncoreMetalMachine readiness criteria are not met.
	IroncoreMetalMachineNotReadyReason = clusterv1.NotReadyReason

	// IroncoreMetalMachineReadyUnknownReason surfaces when at least one IroncoreMetalMachine readiness
	// criteria is unknown and none of the readiness criteria is met.
	IroncoreMetalMachineReadyUnknownReason = clusterv1.ReadyUnknownReason
)

// IroncoreMetalMachine's IPAddressesReady condition and corresponding reasons.
const (
	// IroncoreMetalMachineIPAddressesReadyCondition documents whether the IPAddressClaims for the
	// IroncoreMetalMachine's IPAMConfig entries have been fulfilled by the IPAM provider.
	IroncoreMetalMachineIPAddressesReadyCondition = "IPAddressesReady"

	// IroncoreMetalMachineIPAddressesReadyReason surfaces when all IPAddressClaims are fulfilled.
	IroncoreMetalMachineIPAddressesReadyReason = clusterv1.ReadyReason

	// IroncoreMetalMachineIPAddressesNotRequiredReason surfaces when the IroncoreMetalMachine has no
	// IPAMConfig entries.
	IroncoreMetalMachineIPAddressesNotRequiredReason = "NotRequired"

	// IroncoreMetalMachineIPAddressesWaitingForAllocationReason surfaces when at least one IPAddressClaim
	// has not been fulfilled by the IPAM provider yet.
	IroncoreMetalMachineIPAddressesWaitingForAllocationReason = "WaitingForIPAddressAllocation"

	// IroncoreMetalMachineIPAddressClaimFailedReason surfaces when an IPAddressClaim cannot be used,
	// e.g. because its IPAMConfig entry has no ipamRef or it belongs to another IroncoreMetalMachine.
	IroncoreMetalMachineIPAddressClaimFailedReason = "IPAddressClaimFailed"

	// IroncoreMetalMachineIPAddressesInternalErrorReason surfaces unexpected failures when reading or
	// writing IPAddressClaims or IPAddresses.
	IroncoreMetalMachineIPAddressesInternalErrorReason = clusterv1.InternalErrorReason
)

// IroncoreMetalMachine's ServerClaimReady condition and corresponding reasons.
const (
	// IroncoreMetalMachineServerClaimReadyCondition documents whether the ServerClaim for the
	// IroncoreMetalMachine exists and is bound to a Server.
	IroncoreMetalMachineServerClaimReadyCondition = "ServerClaimReady"

	// IroncoreMetalMachineServerClaimBoundReason surfaces when the ServerClaim is bound to a Server.
	IroncoreMetalMachineServerClaimBoundReason = "Bound"

	// IroncoreMetalMachineServerClaimWaitingForClusterInfrastructureReadyReason surfaces when the
	// Cluster's infrastructure is not provisioned yet.
	IroncoreMetalMachineServerClaimWaitingForClusterInfrastructureReadyReason = clusterv1.WaitingForClusterInfrastructureReadyReason

	// IroncoreMetalMachineServerClaimWaitingForBootstrapDataReason surfaces when the Machine's
	// bootstrap data secret is not available yet.
	IroncoreMetalMachineServerClaimWaitingForBootstrapDataReason = clusterv1.WaitingForBootstrapDataReason

	// IroncoreMetalMachineServerClaimWaitingForIPAddressesReason surfaces when the IP addresses are not
	// ready yet; see the IPAddressesReady condition for details.
	IroncoreMetalMachineServerClaimWaitingForIPAddressesReason = "WaitingForIPAddresses"

	// IroncoreMetalMachineServerClaimWaitingForBoundReason surfaces when the ServerClaim exists but is
	// not bound to a Server yet.
	IroncoreMetalMachineServerClaimWaitingForBoundReason = "WaitingForServerClaimBound"

	// IroncoreMetalMachineServerClaimIgnitionFailedReason surfaces when the ignition cannot be built,
	// e.g. because the bootstrap data or the metadata is not valid JSON.
	IroncoreMetalMachineServerClaimIgnitionFailedReason = "IgnitionFailed"

	// IroncoreMetalMachineServerClaimInternalErrorReason surfaces unexpected failures when reading or
	// writing the bootstrap data, the ignition secret or the ServerClaim.
	IroncoreMetalMachineServerClaimInternalErrorReason = clusterv1.InternalErrorReason
)
