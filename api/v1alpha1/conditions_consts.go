// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	clusterv1b1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

const (
	// IroncoreMetalClusterReady documents the status of IroncoreMetalCluster and its underlying resources.
	IroncoreMetalClusterReady clusterv1b1.ConditionType = "ClusterReady"
)
