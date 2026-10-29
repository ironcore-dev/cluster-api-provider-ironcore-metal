// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/imdario/mergo"
	infrav1alpha1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/ironcore-dev/cluster-api-provider-ironcore-metal/internal/scope"
	metalv1alpha1 "github.com/ironcore-dev/metal-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	clusterapiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/finalizers"
	"sigs.k8s.io/cluster-api/util/paused"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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
	fileSystem                    = "root"
	bootstrapDataKey              = "value"
	metalHostnamePlaceholder      = "%24%24%7BMETAL_HOSTNAME%7D"
	LabelKeyServerClaimName       = "metal.ironcore.dev/server-claim-name"
	LabelKeyServerClaimNamespace  = "metal.ironcore.dev/server-claim-namespace"

	// internalErrorMessage is the message of conditions with an InternalError reason;
	// the error itself is logged.
	internalErrorMessage = "Please check controller logs for errors"
)

// errIPAddressClaimFailed is wrapped by errors caused by the IPAMConfig or by an existing
// IPAddressClaim that cannot be used, as opposed to unexpected API errors.
var errIPAddressClaimFailed = errors.New("IPAddressClaim cannot be used")

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ironcoremetalmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=metal.ironcore.dev,resources=serverclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *IroncoreMetalMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
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

	if metalMachine.Labels[clusterapiv1beta2.ClusterNameLabel] != cluster.Name {
		base := metalMachine.DeepCopy()
		if metalMachine.Labels == nil {
			metalMachine.Labels = map[string]string{}
		}
		metalMachine.Labels[clusterapiv1beta2.ClusterNameLabel] = cluster.Name

		if err := r.Patch(ctx, metalMachine, client.MergeFrom(base)); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, logger)

	// Set the Paused condition and stop reconciling if the object or the owning Cluster is paused
	if isPaused, requeue, err := paused.EnsurePausedCondition(ctx, r.Client, cluster, metalMachine); err != nil || isPaused || requeue {
		return ctrl.Result{}, err
	}

	// Add the finalizer before creating any resources and before the scope's patch helper takes
	// its snapshot, so it is persisted before the first ServerClaim, ignition secret or
	// IPAddressClaim exists. The resulting watch event re-enqueues the object.
	if finalizerAdded, err := finalizers.EnsureFinalizer(ctx, r.Client, metalMachine, IroncoreMetalMachineFinalizer); err != nil || finalizerAdded {
		return ctrl.Result{}, err
	}

	metalClusterName := client.ObjectKey{
		Namespace: metalMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	metalCluster := &infrav1alpha1.IroncoreMetalCluster{}
	if err := r.Get(ctx, metalClusterName, metalCluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("IroncoreMetalCluster is not available yet")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
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
	// A failed patch is returned so the object is reconciled again.
	defer func() {
		if err := machineScope.Close(); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, errors.Wrap(err, "failed to close IroncoreMetalMachine scope")})
		}
	}()

	// Handle deleted machines
	if !metalMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope, clusterScope)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IroncoreMetalMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	log := ctrl.LoggerFrom(ctx)

	clusterToMachines, err := util.ClusterToTypedObjectsMapper(mgr.GetClient(), &infrav1alpha1.IroncoreMetalMachineList{}, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to create Cluster to IroncoreMetalMachines mapper: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.IroncoreMetalMachine{}).
		// ServerClaims are created with a controller reference to the
		// IroncoreMetalMachine, so any change (e.g. becoming Bound) enqueues it.
		Owns(&metalv1alpha1.ServerClaim{}).
		Watches(
			&clusterapiv1beta2.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1alpha1.GroupVersion.WithKind("IroncoreMetalMachine"))),
		).
		// Enqueue all IroncoreMetalMachines of a Cluster when its infrastructure
		// becomes provisioned or it is paused/unpaused.
		Watches(
			&clusterapiv1beta2.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToMachines),
			builder.WithPredicates(predicates.ClusterPausedTransitionsOrInfrastructureProvisioned(mgr.GetScheme(), log)),
		).
		// IPAddressClaims are fulfilled asynchronously by the IPAM provider;
		// enqueue the owning machine when that happens.
		Watches(
			&capiv1beta2.IPAddressClaim{},
			handler.EnqueueRequestsFromMapFunc(ipAddressClaimToMachine),
		).
		Complete(r)
}

func (r *IroncoreMetalMachineReconciler) reconcileDelete(_ context.Context, machineScope *scope.MachineScope) (ctrl.Result, error) {
	machineScope.Info("Deleting IroncoreMetalMachine")

	// Surface the deletion on the ServerClaimReady condition; the Ready summary picks it up.
	conditions.Set(machineScope.IroncoreMetalMachine, metav1.Condition{
		Type:   infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: infrav1alpha1.IroncoreMetalMachineServerClaimDeletingReason,
	})

	// insert ServerClaim deletion logic here

	// The finalizer removal is persisted by the scope's patch helper, like any other change.
	controllerutil.RemoveFinalizer(machineScope.IroncoreMetalMachine, IroncoreMetalMachineFinalizer)
	machineScope.Info("Removed finalizer")

	return ctrl.Result{}, nil
}

func (r *IroncoreMetalMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Logger.V(4).Info("Reconciling IroncoreMetalMachine")

	metalMachine := machineScope.IroncoreMetalMachine

	if !ptr.Deref(machineScope.Cluster.Status.Initialization.InfrastructureProvisioned, false) {
		// No requeue needed: the Cluster watch enqueues this machine once
		// infrastructure becomes provisioned.
		machineScope.Info("Cluster infrastructure is not ready yet")
		conditions.Set(metalMachine, metav1.Condition{
			Type:   infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: infrav1alpha1.IroncoreMetalMachineServerClaimWaitingForClusterInfrastructureReadyReason,
		})
		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		conditions.Set(metalMachine, metav1.Condition{
			Type:   infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: infrav1alpha1.IroncoreMetalMachineServerClaimWaitingForBootstrapDataReason,
		})
		return ctrl.Result{}, nil
	}

	// Fetch the bootstrap data secret.
	bootstrapSecret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: machineScope.Machine.Namespace,
		Name:      *machineScope.Machine.Spec.Bootstrap.DataSecretName,
	}
	if err := r.Get(ctx, secretName, bootstrapSecret); err != nil {
		machineScope.Error(err, "failed to get bootstrap data secret")
		setServerClaimInternalError(metalMachine)
		return ctrl.Result{}, err
	}

	var (
		ipAddressClaims     []*capiv1beta2.IPAddressClaim
		ipAddressesMetadata map[string]any
	)
	if len(metalMachine.Spec.IPAMConfig) == 0 {
		conditions.Set(metalMachine, metav1.Condition{
			Type:   infrav1alpha1.IroncoreMetalMachineIPAddressesReadyCondition,
			Status: metav1.ConditionTrue,
			Reason: infrav1alpha1.IroncoreMetalMachineIPAddressesNotRequiredReason,
		})
	} else {
		var pendingClaim string
		var err error
		ipAddressClaims, ipAddressesMetadata, pendingClaim, err = r.getOrCreateIPAddressClaims(ctx, machineScope.Logger, metalMachine)
		if err != nil {
			machineScope.Error(err, "failed to get or create IPAddressClaims")
			if errors.Is(err, errIPAddressClaimFailed) {
				conditions.Set(metalMachine, metav1.Condition{
					Type:    infrav1alpha1.IroncoreMetalMachineIPAddressesReadyCondition,
					Status:  metav1.ConditionFalse,
					Reason:  infrav1alpha1.IroncoreMetalMachineIPAddressClaimFailedReason,
					Message: err.Error(),
				})
			} else {
				conditions.Set(metalMachine, metav1.Condition{
					Type:    infrav1alpha1.IroncoreMetalMachineIPAddressesReadyCondition,
					Status:  metav1.ConditionUnknown,
					Reason:  infrav1alpha1.IroncoreMetalMachineIPAddressesInternalErrorReason,
					Message: internalErrorMessage,
				})
			}
			setServerClaimWaitingForIPAddresses(metalMachine)
			return ctrl.Result{}, err
		}
		if pendingClaim != "" {
			// No requeue needed: the IPAddressClaim watch enqueues this machine
			// once the IPAM provider has allocated the address.
			machineScope.Info("Waiting for IPAddressClaims to be fulfilled", "IPAddressClaim", pendingClaim)
			conditions.Set(metalMachine, metav1.Condition{
				Type:    infrav1alpha1.IroncoreMetalMachineIPAddressesReadyCondition,
				Status:  metav1.ConditionFalse,
				Reason:  infrav1alpha1.IroncoreMetalMachineIPAddressesWaitingForAllocationReason,
				Message: fmt.Sprintf("Waiting for IPAddressClaim %s to be fulfilled", pendingClaim),
			})
			setServerClaimWaitingForIPAddresses(metalMachine)
			return ctrl.Result{}, nil
		}
		conditions.Set(metalMachine, metav1.Condition{
			Type:   infrav1alpha1.IroncoreMetalMachineIPAddressesReadyCondition,
			Status: metav1.ConditionTrue,
			Reason: infrav1alpha1.IroncoreMetalMachineIPAddressesReadyReason,
		})
	}

	machineScope.Info("Creating an ignition", "Machine", metalMachine.Name)
	ignition, err := r.createIgnition(metalMachine, bootstrapSecret.Data[bootstrapDataKey], ipAddressesMetadata)
	if err != nil {
		machineScope.Error(err, "failed to create an ignition")
		conditions.Set(metalMachine, metav1.Condition{
			Type:    infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  infrav1alpha1.IroncoreMetalMachineServerClaimIgnitionFailedReason,
			Message: err.Error(),
		})
		return ctrl.Result{}, err
	}

	machineScope.Info("Creating IgnitionSecret", "Secret", metalMachine.Name)
	ignitionSecret, err := r.applyIgnitionSecret(ctx, machineScope.Logger, bootstrapSecret, ignition)
	if err != nil {
		machineScope.Error(err, "failed to create or patch ignition secret")
		setServerClaimInternalError(metalMachine)
		return ctrl.Result{}, err
	}

	machineScope.Info("Creating ServerClaim", "ServerClaim", metalMachine.Name)
	serverClaim, err := r.applyServerClaim(ctx, machineScope.Logger, metalMachine, ignitionSecret)
	if err != nil {
		machineScope.Error(err, "failed to create or patch ServerClaim")
		setServerClaimInternalError(metalMachine)
		return ctrl.Result{}, err
	}

	err = r.setServerClaimOwnership(ctx, serverClaim, ipAddressClaims)
	if err != nil {
		machineScope.Error(err, "failed to set ServerClaim ownership")
		setServerClaimInternalError(metalMachine)
		return ctrl.Result{}, err
	}

	currentClaim, err := r.getServerClaim(ctx, serverClaim)
	if err != nil {
		machineScope.Error(err, "failed to check ServerClaim binding")
		setServerClaimInternalError(metalMachine)
		return ctrl.Result{}, err
	}
	if currentClaim == nil || currentClaim.Status.Phase != metalv1alpha1.PhaseBound {
		// No requeue needed: the ServerClaim watch enqueues this machine
		// once the claim's phase changes.
		machineScope.Info("Waiting for ServerClaim to be Bound")
		message := fmt.Sprintf("Waiting for ServerClaim %s to be bound", serverClaim.Name)
		if currentClaim != nil && currentClaim.Status.Phase != "" {
			message = fmt.Sprintf("%s, current phase: %s", message, currentClaim.Status.Phase)
		}
		conditions.Set(metalMachine, metav1.Condition{
			Type:    infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  infrav1alpha1.IroncoreMetalMachineServerClaimWaitingForBoundReason,
			Message: message,
		})
		return ctrl.Result{}, nil
	}

	// The ProviderID is persisted by the scope's patch helper, like the status.
	metalMachine.Spec.ProviderID = fmt.Sprintf("metal://%s/%s", serverClaim.Namespace, serverClaim.Name)
	machineScope.Info("Set ProviderID in IroncoreMetalMachine", "ProviderID", metalMachine.Spec.ProviderID)

	conditions.Set(metalMachine, metav1.Condition{
		Type:   infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
		Status: metav1.ConditionTrue,
		Reason: infrav1alpha1.IroncoreMetalMachineServerClaimBoundReason,
	})
	metalMachine.Status.Ready = true                              // deprecated v1beta1
	metalMachine.Status.Initialization.Provisioned = ptr.To(true) // v1beta2
	machineScope.Info("IroncoreMetalMachine is ready")

	return reconcile.Result{}, nil
}

// setServerClaimWaitingForIPAddresses marks the ServerClaimReady condition as
// waiting for the IP addresses; the IPAddressesReady condition has the details.
func setServerClaimWaitingForIPAddresses(metalMachine *infrav1alpha1.IroncoreMetalMachine) {
	conditions.Set(metalMachine, metav1.Condition{
		Type:    infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  infrav1alpha1.IroncoreMetalMachineServerClaimWaitingForIPAddressesReason,
		Message: "Waiting for IP addresses, see the IPAddressesReady condition",
	})
}

// setServerClaimInternalError marks the ServerClaimReady condition as unknown
// because of an unexpected error; the error itself is logged by the caller.
func setServerClaimInternalError(metalMachine *infrav1alpha1.IroncoreMetalMachine) {
	conditions.Set(metalMachine, metav1.Condition{
		Type:    infrav1alpha1.IroncoreMetalMachineServerClaimReadyCondition,
		Status:  metav1.ConditionUnknown,
		Reason:  infrav1alpha1.IroncoreMetalMachineServerClaimInternalErrorReason,
		Message: internalErrorMessage,
	})
}

func (r *IroncoreMetalMachineReconciler) createIgnition(ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, ignition []byte, ipAddressesMetadata map[string]any) ([]byte, error) {
	ignition = findAndReplaceIgnition(ironcoremetalmachine, ignition)

	ignitionMap := make(map[string]any)
	if err := json.Unmarshal(ignition, &ignitionMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret data: %w", err)
	}

	metaDataMap := make(map[string]any)
	if ironcoremetalmachine.Spec.Metadata != nil {
		if err := json.Unmarshal(ironcoremetalmachine.Spec.Metadata.Raw, &metaDataMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	maps.Copy(metaDataMap, ipAddressesMetadata)
	metaData, err := json.Marshal(metaDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to apply IPAddresses: %w", err)
	}

	if len(metaData) > 2 { // check if metaData is not an empty json object e.g. {}
		metaDataConf := map[string]any{
			"storage": map[string]any{
				"files": []any{map[string]any{
					"path": metaDataFile,
					"mode": fileMode,
					// TODO: remove fileSystem after Ignition 3.x is supported
					"filesystem": fileSystem,
					"contents": map[string]any{
						"compression": "",
						"source":      "data:;base64," + base64.StdEncoding.EncodeToString(metaData),
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

// getOrCreateIPAddressClaims ensures an IPAddressClaim exists for every IPAMConfig entry.
// pendingClaim is the name of the first claim that has not been fulfilled by the IPAM
// provider yet, or empty once all claims are fulfilled; the IPAddressClaim watch
// re-enqueues the machine once it is. Errors caused by the IPAMConfig or by an existing
// claim that cannot be used wrap errIPAddressClaimFailed.
func (r *IroncoreMetalMachineReconciler) getOrCreateIPAddressClaims(ctx context.Context, log *logr.Logger, ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine) (claims []*capiv1beta2.IPAddressClaim, metadata map[string]any, pendingClaim string, err error) {
	claims = []*capiv1beta2.IPAddressClaim{}
	metadata = make(map[string]any)

	for _, networkRef := range ironcoremetalmachine.Spec.IPAMConfig {
		ipAddrClaimName := fmt.Sprintf("%s-%s", ironcoremetalmachine.Name, networkRef.MetadataKey)
		if len(ipAddrClaimName) > validation.DNS1123SubdomainMaxLength {
			log.Info("IP address claim name is too long, it will be shortened which can cause name collisions", "name", ipAddrClaimName)
			ipAddrClaimName = ipAddrClaimName[:validation.DNS1123SubdomainMaxLength]
		}
		ipAddrClaimKey := client.ObjectKey{Namespace: ironcoremetalmachine.Namespace, Name: ipAddrClaimName}

		ipClaim := &capiv1beta2.IPAddressClaim{}
		err := r.Get(ctx, ipAddrClaimKey, ipClaim)
		switch {
		case apierrors.IsNotFound(err):
			if networkRef.IPAMRef == nil {
				return nil, nil, "", fmt.Errorf("%w: ipamRef of ipamConfig %q is not set", errIPAddressClaimFailed, networkRef.MetadataKey)
			}
			log.V(3).Info("Creating IP address claim", "name", ipAddrClaimKey.String())
			ipClaim = &capiv1beta2.IPAddressClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ipAddrClaimKey.Name,
					Namespace: ipAddrClaimKey.Namespace,
					Labels: map[string]string{
						LabelKeyServerClaimName:      ironcoremetalmachine.Name,
						LabelKeyServerClaimNamespace: ironcoremetalmachine.Namespace,
					},
				},
				Spec: capiv1beta2.IPAddressClaimSpec{
					PoolRef: capiv1beta2.IPPoolReference{
						APIGroup: networkRef.IPAMRef.APIGroup,
						Kind:     networkRef.IPAMRef.Kind,
						Name:     networkRef.IPAMRef.Name,
					},
				},
			}
			if err := r.Create(ctx, ipClaim); err != nil {
				return nil, nil, "", fmt.Errorf("error creating IPAddressClaim %q: %w", ipAddrClaimKey.String(), err)
			}
			// Just created: the IPAM provider has not allocated an address yet.
			log.V(3).Info("Waiting for IP address claim to be fulfilled", "name", ipAddrClaimKey.String())
			return nil, nil, ipAddrClaimKey.Name, nil

		case err != nil:
			return nil, nil, "", err
		}

		log.V(3).Info("IP address claim found", "IP", ipAddrClaimKey.String())
		name, nameExists := ipClaim.Labels[LabelKeyServerClaimName]
		namespace, namespaceExists := ipClaim.Labels[LabelKeyServerClaimNamespace]
		if !nameExists || !namespaceExists {
			return nil, nil, "", fmt.Errorf("%w: IP address claim %q has no server claim labels", errIPAddressClaimFailed, ipAddrClaimKey.String())
		}
		if name != ironcoremetalmachine.Name || namespace != ironcoremetalmachine.Namespace {
			return nil, nil, "", fmt.Errorf("%w: IP address claim %q's server claim labels don't match. Expected: name: %q, namespace: %q. Actual: name: %q, namespace: %q", errIPAddressClaimFailed, ipAddrClaimKey.String(), ironcoremetalmachine.Name, ironcoremetalmachine.Namespace, name, namespace)
		}

		if ipClaim.Status.AddressRef.Name == "" {
			log.V(3).Info("Waiting for IP address claim to be fulfilled", "name", ipAddrClaimKey.String())
			return nil, nil, ipAddrClaimKey.Name, nil
		}

		ipAddrKey := client.ObjectKey{Namespace: ipClaim.Namespace, Name: ipClaim.Status.AddressRef.Name}
		ipAddr := &capiv1beta2.IPAddress{}
		if err := r.Get(ctx, ipAddrKey, ipAddr); err != nil {
			return nil, nil, "", err
		}
		ipAddrCopy := ipAddr.DeepCopy()
		if err := controllerutil.SetOwnerReference(ironcoremetalmachine, ipAddr, r.Client.Scheme()); err != nil {
			return nil, nil, "", fmt.Errorf("failed to set OwnerReference: %w", err)
		}
		if err := r.Patch(ctx, ipAddr, client.MergeFrom(ipAddrCopy)); err != nil {
			return nil, nil, "", fmt.Errorf("failed to patch IPAddress: %w", err)
		}

		claims = append(claims, ipClaim)
		metadata[networkRef.MetadataKey] = map[string]any{
			"ip":      ipAddr.Spec.Address,
			"prefix":  ipAddr.Spec.Prefix,
			"gateway": ipAddr.Spec.Gateway,
		}
	}
	return claims, metadata, "", nil
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
			Tolerations:    ironcoremetalmachine.Spec.Tolerations,
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

func (r *IroncoreMetalMachineReconciler) setServerClaimOwnership(ctx context.Context, serverClaim *metalv1alpha1.ServerClaim, IPAddressClaims []*capiv1beta2.IPAddressClaim) error {
	// wait for the server claim to be visible in a cache
	err := wait.PollUntilContextTimeout(
		ctx,
		50*time.Millisecond,
		340*time.Millisecond,
		true,
		func(ctx context.Context) (bool, error) {
			if err := r.Get(ctx, client.ObjectKeyFromObject(serverClaim), serverClaim); err != nil {
				return false, err
			}
			return true, nil
		})
	if err != nil {
		return err
	}

	for _, IPAddressClaim := range IPAddressClaims {
		IPAddressClaimCopy := IPAddressClaim.DeepCopy()
		if err := controllerutil.SetOwnerReference(serverClaim, IPAddressClaim, r.Client.Scheme()); err != nil {
			return fmt.Errorf("failed to set OwnerReference: %w", err)
		}
		if err := r.Patch(ctx, IPAddressClaim, client.MergeFrom(IPAddressClaimCopy)); err != nil {
			return fmt.Errorf("failed to patch IPAddressClaim: %w", err)
		}
	}

	return nil
}

// getServerClaim returns the current state of the ServerClaim, or nil if it is
// not yet visible in the cache; the ServerClaim watch will enqueue us once it is.
func (r *IroncoreMetalMachineReconciler) getServerClaim(ctx context.Context, serverClaim *metalv1alpha1.ServerClaim) (*metalv1alpha1.ServerClaim, error) {
	claimObj := &metalv1alpha1.ServerClaim{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(serverClaim), claimObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return claimObj, nil
}

func findAndReplaceIgnition(ironcoremetalmachine *infrav1alpha1.IroncoreMetalMachine, data []byte) []byte {
	// replace $${METAL_HOSTNAME} with machine name
	modifiedData := strings.ReplaceAll(string(data), metalHostnamePlaceholder, ironcoremetalmachine.Name)

	return []byte(modifiedData)
}

// ipAddressClaimToMachine maps an IPAddressClaim created by this controller to
// its IroncoreMetalMachine via the labels set when the claim was created.
func ipAddressClaimToMachine(_ context.Context, o client.Object) []reconcile.Request {
	labels := o.GetLabels()
	name, ok := labels[LabelKeyServerClaimName]
	if !ok {
		return nil
	}
	namespace, ok := labels[LabelKeyServerClaimNamespace]
	if !ok {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}}}
}
