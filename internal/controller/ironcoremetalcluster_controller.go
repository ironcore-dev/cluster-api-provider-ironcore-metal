// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/cluster-api-provider-ironcore-metal/internal/scope"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
)

// IroncoreMetalClusterReconciler reconciles a IroncoreMetalCluster object
type IroncoreMetalClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *IroncoreMetalClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	metalCluster := &infrav1.IroncoreMetalCluster{}
	if err := r.Get(ctx, req.NamespacedName, metalCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get owner cluster
	cluster, err := util.GetOwnerCluster(ctx, r.Client, metalCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		logger.Info("waiting for Cluster Controller to set OwnerRef on IroncoreMetalCluster")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, logger)

	if annotations.IsPaused(cluster, metalCluster) {
		logger.Info("IroncoreMetalCluster or owning Cluster is marked as paused, not reconciling")
		return ctrl.Result{}, nil
	}

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               &logger,
		Cluster:              cluster,
		IroncoreMetalCluster: metalCluster,
		ControllerName:       "ironcoremetalcluster",
	})

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function, so we can persist any IroncoreMetalCluster changes.
	defer func() {
		if err := clusterScope.Close(); err != nil {
			logger.Error(err, "failed to close IroncoreMetalCluster scope")
		}
	}()

	// Handle deleted clusters
	if !metalCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, clusterScope)

}

func (r *IroncoreMetalClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	// We want to prevent deletion unless the owning cluster was flagged for deletion.
	if clusterScope.Cluster.DeletionTimestamp.IsZero() {
		clusterScope.Error(errors.New("deletion was requested but owning cluster wasn't deleted"), "Unable to delete IroncoreMetalCluster")
		// We stop reconciling here. It will be triggered again once the owning cluster was deleted.
		return reconcile.Result{}, nil
	}

	clusterScope.Logger.V(4).Info("reconciling IroncoreMetalCluster delete")
	// Deletion usually should be triggered through the deletion of the owning cluster.
	// If the IroncoreMetalCluster was also flagged for deletion (e.g. deletion using the manifest file)
	// we should only allow to remove the finalizer when there are no IroncoreMetalMachines left.
	machines, err := r.listIroncoreMetalMachinesForCluster(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "could not retrieve metal machines for cluster %q", clusterScope.InfraClusterName())
	}

	// Requeue if there are one or more machines left.
	if len(machines) > 0 {
		clusterScope.Info("waiting for machines to be deleted", "remaining", len(machines))
		return ctrl.Result{RequeueAfter: infrav1.DefaultReconcilerRequeue}, nil
	}

	clusterScope.Info("cluster deleted successfully")
	ctrlutil.RemoveFinalizer(clusterScope.IroncoreMetalCluster, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

//nolint:unparam
func (r *IroncoreMetalClusterReconciler) reconcileNormal(_ context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling IroncoreMetalCluster")

	// If the IroncoreMetalCluster doesn't have our finalizer, add it.
	ctrlutil.AddFinalizer(clusterScope.IroncoreMetalCluster, infrav1.ClusterFinalizer)

	conditions.Set(clusterScope.IroncoreMetalCluster, metav1.Condition{
		Type:    infrav1.IroncoreMetalClusterReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "IronMetalCluster is ready",
	})

	clusterScope.IroncoreMetalCluster.Status.Ready = true

	return ctrl.Result{}, nil
}

func (r *IroncoreMetalClusterReconciler) listIroncoreMetalMachinesForCluster(ctx context.Context, clusterScope *scope.ClusterScope) ([]infrav1.IroncoreMetalMachine, error) {
	var machineList infrav1.IroncoreMetalMachineList
	err := r.List(ctx, &machineList, client.InNamespace(clusterScope.Namespace()), client.MatchingLabels{
		clusterv1.ClusterNameLabel: clusterScope.Name(),
	})
	if err != nil {
		return nil, err
	}
	fmt.Println("listing machines", clusterScope.Name(), machineList.Items)
	return machineList.Items, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IroncoreMetalClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.IroncoreMetalCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(ctx, infrav1.GroupVersion.WithKind("IroncoreMetalCluster"), mgr.GetClient(), &infrav1.IroncoreMetalCluster{})),
		).
		Complete(r)
}
