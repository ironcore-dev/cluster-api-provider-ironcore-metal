//go:build e2e

// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

type TestContext struct {
	Settings    Settings
	E2EConfig   *clusterctl.E2EConfig
	Environment RuntimeEnvironment
}

type Settings struct {
	ConfigPath     string
	ArtifactFolder string
	DataFolder     string
	SkipCleanup    bool
}

type RuntimeEnvironment struct {
	ClusterctlConfigPath     string
	Scheme                   *runtime.Scheme
}