// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/imdario/mergo"
	infrav1alpha1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/ironcore-dev/cluster-api-provider-ironcore-metal/internal/scope"
	"github.com/ironcore-dev/controller-utils/clientutils"
	metalv1alpha1 "github.com/ironcore-dev/metal-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	clusterapiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiv1beta1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IroncoreMetalMachineReconciler reconciles a IroncoreMetalMachine object
type IroncoreMetalMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	IroncoreMetalMachineFinalizer = "infrastructure.cluster.x-k8s.io/ironcoremetalmachine"
	DefaultIgnitionSecretKeyName  = "ignition"
	metaDataFile                  = "/var/lib/metal-cloud-config/metadata"
	fileMode                      = 0644
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=metal.ironcore.dev,resources=serverclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *IroncoreMetalMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the IroncoreMetalMachine.
	metalMachine := &infrav1alpha1.IroncoreMetalMachine{}
	err := r.Get(ctx, req.NamespacedName, metalMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, metalMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", klog.KObj(machine))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, metalMachine) {
		logger.Info("IroncoreMetalMachine or linked Cluster is marked as paused, not reconciling")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))

	metalClusterName := client.ObjectKey{
		Namespace: metalMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	metalCluster := &infrav1alpha1.IroncoreMetalCluster{}
	if err := r.Client.Get(ctx, metalClusterName, metalCluster); err != nil {
		if apierrors.IsNotFound(err) || !metalCluster.Status.Ready {
			logger.Info("IroncoreMetalCluster is not available yet")
			return ctrl.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               &logger,
		Cluster:              cluster,
		IroncoreMetalCluster: metalCluster,
		ControllerName:       "ironcoremetalcluster",
	})

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create cluster scope: %+v", err)
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:               r.Client,
		Cluster:              cluster,
		Machine:              machine,
		IroncoreMetalCluster: metalCluster,
		IroncoreMetalMachine: metalMachine,
	})

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create machine scope: %+v", err)
	}

	// Always close the scope when exiting this function, so we can persist any IroncoreMetalMachine changes.
	// TODO: revisit side effects of closure errors
	defer func() {
		if err := machineScope.Close(); err != nil {
			logger.Error(err, "failed to close IroncoreMetalMachine scope")
		}
	}()

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, metalMachine) {
		logger.Info("IroncoreMetalMachine or linked Cluster is marked as paused. Won't reconcile normally")
		return reconcile.Result{}, nil
	}

	// Handle deleted machines
	if !metalMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope, clusterScope)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IroncoreMetalMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.IroncoreMetalMachine{}).
		Watches(
			&clusterapiv1beta1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1alpha1.GroupVersion.WithKind("IroncoreMetalMachine"))),
		).
		Complete(r)
}

func (r *IroncoreMetalMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope) (ctrl.Result, error) {
	machineScope.Logger.Info("Deleting IroncoreMetalMachine")

	// insert ServerClaim deletion logic here

	ip := &capiv1beta1.IPAddressClaim{ObjectMeta: metav1.ObjectMeta{
		Name:      machineScope.IroncoreMetalMachine.Name,
		Namespace: machineScope.IroncoreMetalMachine.Namespace,
	}}
	if err := r.Client.Delete(ctx, ip); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting ip resource: %s", err.Error())
	}

	if modified, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, machineScope.IroncoreMetalMachine, IroncoreMetalMachineFinalizer); !apierrors.IsNotFound(err) || modified {
		return ctrl.Result{}, err
	}
	machineScope.Logger.Info("Ensured that the finalizer has been removed")

	return reconcile.Result{RequeueAfter: infrav1alpha1.DefaultReconcilerRequeue}, nil
}

func (r *IroncoreMetalMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Logger.V(4).Info("Reconciling IroncoreMetalMachine")

	// If the IroncoreMetalMachine is in an error state, return early.
	if machineScope.HasFailed() {
		machineScope.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		// TBD: update conditions
		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		// TBD: update conditions
		return ctrl.Result{}, nil
	}

	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, machineScope.IroncoreMetalMachine, IroncoreMetalMachineFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}
	machineScope.Logger.Info("Ensured finalizer has been added")

	// Fetch the bootstrap data secret.
	bootstrapSecret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: machineScope.Machine.Namespace,
		Name:      *machineScope.Machine.Spec.Bootstrap.DataSecretName,
	}
	if err := r.Client.Get(ctx, secretName, bootstrapSecret); err != nil {
		machineScope.Error(err, "failed to get bootstrap data secret")
		return ctrl.Result{}, err
	}

	machineScope.Info("Creating secret data", "Secret", machineScope.IroncoreMetalMachine.Name)
	ignition, err := r.createIgnition(ctx, machineScope.Logger, machineScope.IroncoreMetalMachine, bootstrapSecret.Data["value"])
	if err != nil {
		machineScope.Error(err, "failed to create or patch ignition secret")
		return ctrl.Result{}, err
	}

	machineScope.Info("Creating IgnitionSecret", "Secret", machineScope.IroncoreMetalMachine.Name)
	ignitionSecret, err := r.applyIgnitionSecret(ctx, machineScope.Logger, bootstrapSecret, ignition)
	if err != nil {
		machineScope.Error(err, "failed to create or patch ignition secret")
		return ctrl.Result{}, err
	}

	machineScope.Info("Creating ServerClaim", "ServerClaim", machineScope.IroncoreMetalMachine.Name)
	serverClaim, err := r.applyServerClaim(ctx, machineScope.Logger, machineScope.IroncoreMetalMachine, ignitionSecret)
	if err != nil {
		machineScope.Error(err, "failed to create or patch ServerClaim")
		return ctrl.Result{}, err
	}

	bound, _ := r.ensureServerClaimBound(ctx, serverClaim)
	if !bound {
		machineScope.Info("Waiting for ServerClaim to be Bound")
		return ctrl.Result{
			RequeueAfter: infrav1alpha1.DefaultReconcilerRequeue,
		}, nil
	}

	machineScope.Info("Patching ProviderID in IroncoreMetalMachine")
	if err := r.patchIroncoreMetalMachineProviderID(ctx, machineScope.Logger, machineScope.IroncoreMetalMachine, serverClaim); err != nil {
		machineScope.Error(err, "failed to patch the IroncoreMetalMachine with providerid")
		return ctrl.Result{}, err
	}

	machineScope.SetReady()
	machineScope.Logger.Info("IroncoreMetalMachine is ready")

	return reconcile.Result{}, nil
}

func (r *IroncoreMetalMachineReconciler) createIgnition(ctx context.Context, log *logr.Logger, ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, ignition []byte) ([]byte, error) {
	ignition = findAndReplaceIgnition(ironcoremetalmachine, ignition)

	ignitionMap := make(map[string]any)
	if err := json.Unmarshal(ignition, &ignitionMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret data: %w", err)
	}

	metaData, err := r.createMetaData(ctx, log, ironcoremetalmachine, ironcoremetalmachine.Spec.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to apply IPAddresses: %w", err)
	}

	if len(metaData) > 0 {
		metaDataConf := map[string]any{
			"storage": map[string]any{
				"files": []any{map[string]any{
					"path": metaDataFile,
					"mode": fileMode,
					"contents": map[string]any{
						"inline": string(metaData),
					},
				}},
			},
		}
		// merge metaData configuration with ignition content
		if err := mergo.Merge(&ignitionMap, metaDataConf, mergo.WithAppendSlice); err != nil {
			return nil, fmt.Errorf("failed to merge metaData configuration with ignition content: %w", err)
		}
	}

	return json.Marshal(ignitionMap)
}

func (r *IroncoreMetalMachineReconciler) createMetaData(ctx context.Context, log *logr.Logger, ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, metaData *apiextensionsv1.JSON) ([]byte, error) {
	metaDataMap := make(map[string]any)
	if metaData != nil {
		if err := json.Unmarshal(metaData.Raw, &metaDataMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	for _, networkRef := range ironcoremetalmachine.Spec.IPAMConfig {
		ipAddrKey := client.ObjectKeyFromObject(ironcoremetalmachine)
		ipClaim := &capiv1beta1.IPAddressClaim{}
		if err := r.Client.Get(ctx, ipAddrKey, ipClaim); err != nil && !apierrors.IsNotFound(err) {
			return nil, err

		} else if err == nil {
			log.V(3).Info("IP found", "IP", ipAddrKey.String())
			if ipClaim.Status.AddressRef.Name == "" {
				return nil, errors.New("IP address claim isn't ready")
			}

		} else if apierrors.IsNotFound(err) {
			if networkRef.IPAMRef == nil {
				return nil, errors.New("ipamRef of an ipamConfig is not set")
			}
			log.V(3).Info("creating IP address claim", "name", ipAddrKey.String())
			apiGroup := capiv1beta1.GroupVersion.Group
			ipClaim = &capiv1beta1.IPAddressClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ipAddrKey.Name,
					Namespace: ipAddrKey.Namespace,
				},
				Spec: capiv1beta1.IPAddressClaimSpec{
					PoolRef: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "GlobalInClusterIPPool",
						Name:     networkRef.IPAMRef.Name,
					},
				},
			}
			if err = r.Client.Create(ctx, ipClaim); err != nil {
				return nil, fmt.Errorf("error creating IP: %w", err)
			}

			// Wait for the IP address claim to reach the ready state
			err = wait.PollUntilContextTimeout(
				ctx,
				time.Millisecond*50,
				time.Millisecond*340,
				true,
				func(ctx context.Context) (bool, error) {
					if err = r.Client.Get(ctx, ipAddrKey, ipClaim); err != nil && !apierrors.IsNotFound(err) {
						return false, err
					}
					return ipClaim.Status.AddressRef.Name != "", nil
				})
			if err != nil {
				return nil, err
			}
		}

		ipAddr := &capiv1beta1.IPAddress{}
		if err := r.Client.Get(ctx, ipAddrKey, ipAddr); err != nil {
			return nil, err
		}
		metaDataMap[networkRef.MetadataKey] = map[string]any{
			"ip":      ipAddr.Spec.Address,
			"prefix":  ipAddr.Spec.Prefix,
			"gateway": ipAddr.Spec.Gateway,
		}
	}
	return json.Marshal(metaDataMap)
}

func (r *IroncoreMetalMachineReconciler) applyIgnitionSecret(ctx context.Context, log *logr.Logger, capidatasecret *corev1.Secret, ignition []byte) (*corev1.Secret, error) {
	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ignition-%s", capidatasecret.Name),
			Namespace: capidatasecret.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		Data: map[string][]byte{
			DefaultIgnitionSecretKeyName: ignition,
		},
	}

	if err := controllerutil.SetControllerReference(capidatasecret, secretObj, r.Client.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set ControllerReference: %w", err)
	}

	opResult, err := controllerutil.CreateOrPatch(ctx, r.Client, secretObj, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create or patch the IgnitionSecret: %w", err)
	}
	log.Info("Created or Patched IgnitionSecret", "IgnitionSecret", secretObj.Name, "Operation", opResult)

	return secretObj, nil
}

func (r *IroncoreMetalMachineReconciler) applyServerClaim(ctx context.Context, log *logr.Logger, ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, ignitionsecret *corev1.Secret) (*metalv1alpha1.ServerClaim, error) {
	serverClaimObj := &metalv1alpha1.ServerClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ironcoremetalmachine.Name,
			Namespace: ironcoremetalmachine.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalv1alpha1.GroupVersion.String(),
			Kind:       "ServerClaim",
		},
		Spec: metalv1alpha1.ServerClaimSpec{
			Power: metalv1alpha1.PowerOn,
			IgnitionSecretRef: &corev1.LocalObjectReference{
				Name: ignitionsecret.Name,
			},
			Image:          ironcoremetalmachine.Spec.Image,
			ServerSelector: ironcoremetalmachine.Spec.ServerSelector,
		},
	}

	if err := controllerutil.SetControllerReference(ironcoremetalmachine, serverClaimObj, r.Client.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set ControllerReference: %w", err)
	}

	opResult, err := controllerutil.CreateOrPatch(ctx, r.Client, serverClaimObj, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create or patch ServerClaim: %w", err)
	}
	log.Info("Created or Patched ServerClaim", "ServerClaim", serverClaimObj.Name, "Operation", opResult)

	return serverClaimObj, nil
}

func (r *IroncoreMetalMachineReconciler) patchIroncoreMetalMachineProviderID(ctx context.Context, log *logr.Logger, ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, serverClaim *metalv1alpha1.ServerClaim) error {
	providerID := fmt.Sprintf("metal://%s/%s", serverClaim.Namespace, serverClaim.Name)

	patch := client.MergeFrom(ironcoremetalmachine.DeepCopy())
	ironcoremetalmachine.Spec.ProviderID = &providerID

	if err := r.Client.Patch(ctx, ironcoremetalmachine, patch); err != nil {
		log.Error(err, "failed to patch IroncoreMetalMachine with ProviderID")
		return err
	}

	log.Info("Successfully patched IroncoreMetalMachine with ProviderID", "ProviderID", providerID)
	return nil
}

func (r *IroncoreMetalMachineReconciler) ensureServerClaimBound(ctx context.Context, serverClaim *metalv1alpha1.ServerClaim) (bool, error) {
	claimObj := &metalv1alpha1.ServerClaim{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(serverClaim), claimObj); err != nil {
		return false, err
	}

	if claimObj.Status.Phase != metalv1alpha1.PhaseBound {
		return false, nil
	}
	return true, nil
}

func findAndReplaceIgnition(ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, data []byte) []byte {
	// replace $${METAL_HOSTNAME} with machine name
	hostname := "%24%24%7BMETAL_HOSTNAME%7D"
	modifiedData := strings.ReplaceAll(string(data), hostname, ironcoremetalmachine.Name)

	return []byte(modifiedData)
}
