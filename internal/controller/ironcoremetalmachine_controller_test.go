// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"

	infrav1alpha1 "github.com/ironcore-dev/cluster-api-provider-ironcore-metal/api/v1alpha1"
	"github.com/ironcore-dev/controller-utils/clientutils"
	metalv1alpha1 "github.com/ironcore-dev/metal-operator/api/v1alpha1"
)

var _ = Describe("IroncoreMetalMachine Controller", func() {
	When("all resources are present to reconcile", func() {
		const namespace = "default"

		var (
			ctx                  = context.Background()
			secret               *corev1.Secret
			metalCluster         *infrav1alpha1.IroncoreMetalCluster
			cluster              *clusterapiv1beta2.Cluster
			machine              *clusterapiv1beta2.Machine
			metalMachine         *infrav1alpha1.IroncoreMetalMachine
			controllerReconciler *IroncoreMetalMachineReconciler
			metalSecretNN        = types.NamespacedName{}

			get = func(obj client.Object) error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
			}

			expectIgnition = func(expected string) {
				metalSecret := &corev1.Secret{}
				Expect(k8sClient.Get(ctx, metalSecretNN, metalSecret)).To(Succeed())
				Expect(metalSecret.Data).To(HaveKey(DefaultIgnitionSecretKeyName))
				Expect(metalSecret.Data[DefaultIgnitionSecretKeyName]).To(Equal([]byte(expected)))
			}

			getOwnerReferences = func(obj client.Object) []metav1.OwnerReference {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(Succeed())
				return obj.GetOwnerReferences()
			}
		)

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					bootstrapDataKey: []byte(fmt.Sprintf(`{"name": "%s"}`, metalHostnamePlaceholder)),
				},
			}

			metalCluster = &infrav1alpha1.IroncoreMetalCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metal-cluster",
					Namespace: namespace,
				},
				Spec: infrav1alpha1.IroncoreMetalClusterSpec{
					ControlPlaneEndpoint: clusterapiv1beta2.APIEndpoint{
						Host: "1.2.3.4",
					},
					ClusterNetwork: clusterapiv1beta2.ClusterNetwork{
						ServiceDomain: "test.domain",
					},
				},
			}

			cluster = &clusterapiv1beta2.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster",
					Namespace: namespace,
				},
				Spec: clusterapiv1beta2.ClusterSpec{
					InfrastructureRef: clusterapiv1beta2.ContractVersionedObjectReference{
						APIGroup: infrav1alpha1.GroupVersion.Group,
						Kind:     "IroncoreMetalCluster",
						Name:     metalCluster.Name,
					},
				},
			}

			machine = &clusterapiv1beta2.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine",
					Namespace: namespace,
					Labels:    map[string]string{clusterapiv1beta2.ClusterNameLabel: cluster.Name},
				},
				Spec: clusterapiv1beta2.MachineSpec{
					ClusterName: cluster.Name,
					Bootstrap: clusterapiv1beta2.Bootstrap{
						DataSecretName: &secret.Name,
					},
					InfrastructureRef: clusterapiv1beta2.ContractVersionedObjectReference{
						Kind:     "IroncoreMetalMachine",
						Name:     "test-capi-machine",
						APIGroup: "infrastructure.cluster.x-k8s.io",
					},
				},
			}

			metalMachine = &infrav1alpha1.IroncoreMetalMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metal-machine",
					Namespace: namespace,
				},
			}

			metalSecretNN = types.NamespacedName{Name: fmt.Sprintf("ignition-%s", secret.Name), Namespace: namespace}

			controllerReconciler = &IroncoreMetalMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			Expect(k8sClient.Create(ctx, metalCluster)).To(Succeed())
			Eventually(func() error {
				if err := get(metalCluster); err != nil {
					return err
				}
				metalCluster.Status.Ready = true
				return k8sClient.Status().Update(ctx, metalCluster)
			}).Should(Succeed())
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
			Eventually(func() error {
				if err := get(cluster); err != nil {
					return err
				}
				infraProvisionedFlag := true
				cluster.Status.Initialization.InfrastructureProvisioned = &infraProvisionedFlag
				return k8sClient.Status().Update(ctx, cluster)
			}).Should(Succeed())
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			Expect(controllerutil.SetOwnerReference(machine, metalMachine, k8sClient.Scheme())).To(Succeed())
			Expect(k8sClient.Create(ctx, metalMachine)).To(Succeed())
			Eventually(func() error {
				if err := get(metalMachine); err != nil {
					return err
				}
				return clientutils.PatchAddFinalizer(ctx, k8sClient, metalMachine, IroncoreMetalMachineFinalizer)
			}).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, metalCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).To(Succeed())
			if err := get(metalMachine); err == nil {
				Expect(clientutils.PatchRemoveFinalizer(ctx, k8sClient, metalMachine, IroncoreMetalMachineFinalizer)).To(Succeed())
				Expect(k8sClient.Delete(ctx, metalMachine)).To(Succeed())

				serverClaim := &metalv1alpha1.ServerClaim{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), serverClaim)).To(Succeed())
				Expect(k8sClient.Delete(ctx, serverClaim)).To(Succeed())

				metalSecret := &corev1.Secret{}
				Expect(k8sClient.Get(ctx, metalSecretNN, metalSecret)).To(Succeed())
				Expect(k8sClient.Delete(ctx, metalSecret)).To(Succeed())
			}
		})

		It("should create the ignition secret", func() {
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(metalMachine),
			})
			Expect(err).NotTo(HaveOccurred())

			expectIgnition(`{"name":"metal-machine"}`)
		})

		When("the metadata is present in the metal machine", func() {
			BeforeEach(func() {
				metalMachine.Spec.Metadata = &apiextensionsv1.JSON{
					Raw: []byte(`{"foo": "bar"}`),
				}
			})

			It("should create the ignition secret with the metadata", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())

				ign := base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar"}`))
				expectIgnition(
					`{"name":"metal-machine","storage":{"files":[{"contents":{"compression":"","source":"data:;base64,` +
						ign + `"},"filesystem":"root","mode":420,"path":"/var/lib/metal-cloud-config/metadata"}]}}`)
			})
		})
		When("delete machine", func() {
			It("should delete", func() {
				Expect(k8sClient.Delete(ctx, metalMachine)).To(Succeed())
				result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(metalMachine)})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeZero())
				Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), metalMachine)

				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})
		When("the ipam config is present in the metal machine", func() {
			const metadataKey = "meta-key"

			var (
				ipAddressClaim *capiv1beta2.IPAddressClaim
				ipAddress      *capiv1beta2.IPAddress
			)

			BeforeEach(func() {
				ipAddressClaim = &capiv1beta2.IPAddressClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", metalMachine.Name, metadataKey),
						Namespace: namespace,
					},
				}

				prefix := int32(24)
				ipAddress = &capiv1beta2.IPAddress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ip-address",
						Namespace: namespace,
					},
					Spec: capiv1beta2.IPAddressSpec{
						Address: "10.11.12.13",
						Prefix:  &prefix,
						Gateway: "10.11.12.1",
						ClaimRef: capiv1beta2.IPAddressClaimReference{
							Name: ipAddressClaim.Name,
						},
						PoolRef: capiv1beta2.IPPoolReference{
							Name:     "test-ip-pool",
							Kind:     "GlobalInClusterIPPool",
							APIGroup: "ipam.cluster.x-k8s.io",
						},
					},
				}

				metalMachine.Spec.IPAMConfig = []infrav1alpha1.IPAMConfig{{
					MetadataKey: metadataKey,
					IPAMRef: &infrav1alpha1.IPAMObjectReference{
						Name:     "pool",
						APIGroup: "ipam.cluster.x-k8s.io",
						Kind:     "GlobalInClusterIPPool",
					},
				}}

				Expect(k8sClient.Create(ctx, ipAddress)).To(Succeed())
				go func() {
					defer GinkgoRecover()
					Eventually(UpdateStatus(ipAddressClaim, func() {
						ipAddressClaim.Status.AddressRef.Name = ipAddress.Name
					})).Should(Succeed())
				}()
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, ipAddress)).To(Succeed())
				Expect(k8sClient.Delete(ctx, ipAddressClaim)).To(Succeed())
			})

			It("should create the ignition secret with the ip address", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())

				ign := base64.StdEncoding.EncodeToString([]byte(`{"meta-key":{"gateway":"10.11.12.1","ip":"10.11.12.13","prefix":24}}`))
				expectIgnition(
					`{"name":"metal-machine","storage":{"files":[{"contents":{"compression":"","source":"data:;base64,` +
						ign + `"},"filesystem":"root","mode":420,"path":"/var/lib/metal-cloud-config/metadata"}]}}`)
			})

			It("should set the owner reference on the ip address claim", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())

				serverClaim := &metalv1alpha1.ServerClaim{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), serverClaim)).To(Succeed())
				Eventually(Object(ipAddressClaim)).Should(SatisfyAll(
					HaveField("Labels", HaveKeyWithValue(LabelKeyServerClaimName, serverClaim.Name)),
					HaveField("Labels", HaveKeyWithValue(LabelKeyServerClaimNamespace, serverClaim.Namespace)),
					HaveField("OwnerReferences", ContainElement(
						metav1.OwnerReference{
							APIVersion: metalv1alpha1.GroupVersion.String(),
							Kind:       "ServerClaim",
							Name:       serverClaim.Name,
							UID:        serverClaim.UID,
						},
					))))
			})

			It("should set the owner reference on the ip address", func() {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() []metav1.OwnerReference {
					return getOwnerReferences(ipAddress)
				}).Should(ContainElement(metav1.OwnerReference{
					APIVersion: infrav1alpha1.GroupVersion.String(),
					Kind:       "IroncoreMetalMachine",
					Name:       metalMachine.Name,
					UID:        metalMachine.UID,
				}))
			})

			When("the metadata is present in the metal machine", func() {
				BeforeEach(func() {
					metalMachine.Spec.Metadata = &apiextensionsv1.JSON{
						Raw: []byte(`{"foo": "bar"}`),
					}
				})

				It("should create the ignition secret with the ip address and the metadata", func() {
					_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: client.ObjectKeyFromObject(metalMachine),
					})
					Expect(err).NotTo(HaveOccurred())

					ign := base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar","meta-key":{"gateway":"10.11.12.1","ip":"10.11.12.13","prefix":24}}`))
					expectIgnition(
						`{"name":"metal-machine","storage":{"files":[{"contents":{"compression":"","source":"data:;base64,` +
							ign + `"},"filesystem":"root","mode":420,"path":"/var/lib/metal-cloud-config/metadata"}]}}`)
				})
			})
			It("should set ProviderID and Ready status when ServerClaim is bound", func() {
				// 1st call to create server claim
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())

				// get created server claim to then bound it
				serverClaim := &metalv1alpha1.ServerClaim{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), serverClaim)).To(Succeed())

				// bound it ( in real cluster metal-operator does this)
				serverClaim.Status.Phase = metalv1alpha1.PhaseBound
				Expect(k8sClient.Status().Update(ctx, serverClaim)).To(Succeed())

				// 2nd call - now controller can see that ServerClaim is bound
				out, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(Equal(ctrl.Result{}))

				// fetch updated machine
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), metalMachine)).To(Succeed())

				// build expected like here https://github.com/ironcore-dev/cluster-api-provider-ironcore-metal/blob/main/internal/controller/ironcoremetalmachine_controller.go#L492
				expectedProviderID := fmt.Sprintf("metal://%s/%s", serverClaim.Namespace, serverClaim.Name)

				Eventually(Object(metalMachine)).Should(HaveField("Spec.ProviderID", Equal(expectedProviderID)))

				// check status for v1beta1 contract (Deprecated, but we still use it)
				Eventually(Object(metalMachine)).Should(HaveField("Status.Ready", BeTrue()))

				// check status for v1beta2 contract
				Expect(metalMachine.Status.Initialization).NotTo(BeNil())
				Expect(*metalMachine.Status.Initialization.Provisioned).To(BeTrue())
			})
		})
	})
})

var _ = Describe("IroncoreMetalMachine Controller", func() {
	When("not all resources are present to reconcile and fail", func() {
		const namespace = "default"

		var (
			ctx                  = context.Background()
			secret               *corev1.Secret
			metalCluster         *infrav1alpha1.IroncoreMetalCluster
			cluster              *clusterapiv1beta2.Cluster
			machine              *clusterapiv1beta2.Machine
			metalMachine         *infrav1alpha1.IroncoreMetalMachine
			metalMachineOwner    *infrav1alpha1.IroncoreMetalMachine
			controllerReconciler *IroncoreMetalMachineReconciler
			createOnce           bool
		)

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-test",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					bootstrapDataKey: []byte(fmt.Sprintf(`{"name": "%s"}`, metalHostnamePlaceholder)),
				},
			}
			metalCluster = &infrav1alpha1.IroncoreMetalCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metal-cluster",
					Namespace: namespace,
				},
				Spec: infrav1alpha1.IroncoreMetalClusterSpec{
					ControlPlaneEndpoint: clusterapiv1beta2.APIEndpoint{
						Host: "1.2.3.4",
					},
					ClusterNetwork: clusterapiv1beta2.ClusterNetwork{
						ServiceDomain: "test.domain",
					},
				},
			}

			cluster = &clusterapiv1beta2.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster",
					Namespace: namespace,
				},
				Spec: clusterapiv1beta2.ClusterSpec{
					InfrastructureRef: clusterapiv1beta2.ContractVersionedObjectReference{
						APIGroup: infrav1alpha1.GroupVersion.Group,
						Kind:     "IroncoreMetalCluster",
						Name:     metalCluster.Name,
					},
				},
			}

			machine = &clusterapiv1beta2.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-test",
					Namespace: namespace,
					Labels:    map[string]string{clusterapiv1beta2.ClusterNameLabel: cluster.Name},
				},
				Spec: clusterapiv1beta2.MachineSpec{
					ClusterName: cluster.Name,
					Bootstrap: clusterapiv1beta2.Bootstrap{
						DataSecretName: &secret.Name,
					},
					InfrastructureRef: clusterapiv1beta2.ContractVersionedObjectReference{
						Kind:     "IroncoreMetalMachine",
						Name:     "test-capi-machine2",
						APIGroup: "infrastructure.cluster.x-k8s.io",
					},
				},
			}

			metalMachineOwner = &infrav1alpha1.IroncoreMetalMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metal-machine-test-owner",
					Namespace: namespace,
				},
			}
			metalMachine = &infrav1alpha1.IroncoreMetalMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metal-machine-test",
					Namespace: namespace,
				},
			}
			controllerReconciler = &IroncoreMetalMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			Expect(k8sClient.Create(ctx, metalCluster)).To(Succeed())
			Eventually(UpdateStatus(metalCluster, func() {
				metalCluster.Status.Ready = true
			})).Should(Succeed())
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
			Eventually(UpdateStatus(cluster, func() {
				infraProvisionedFlag := true
				cluster.Status.Initialization.InfrastructureProvisioned = &infraProvisionedFlag
			})).Should(Succeed())
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			Expect(controllerutil.SetOwnerReference(machine, metalMachine, k8sClient.Scheme())).To(Succeed())
			if !createOnce {
				Expect(k8sClient.Create(ctx, metalMachine)).To(Succeed())
				Eventually(func() error {
					return clientutils.PatchAddFinalizer(ctx, k8sClient, metalMachine, IroncoreMetalMachineFinalizer)
				}).Should(Succeed())
				Expect(controllerutil.SetOwnerReference(machine, metalMachineOwner, k8sClient.Scheme())).To(Succeed())
				Expect(k8sClient.Create(ctx, metalMachineOwner)).To(Succeed())
				Eventually(func() error {
					return clientutils.PatchAddFinalizer(ctx, k8sClient, metalMachineOwner, IroncoreMetalMachineFinalizer)
				}).Should(Succeed())
				createOnce = true
			}
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, metalCluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).To(Succeed())
		})
		When("no owner set", func() {
			It("should pass with empty", func() {

				Eventually(Update(metalMachineOwner, func() {
					metalMachineOwner.OwnerReferences = []metav1.OwnerReference{}
				})).To(Succeed())
				out, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachineOwner),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(Equal(ctrl.Result{}))
				Expect(k8sClient.Delete(ctx, metalMachineOwner)).To(Succeed())
			})
		})
		When("no cluster label", func() {
			It("should return empty ", func() {
				Eventually(Update(machine, func() {
					machine.Labels = map[string]string{}
				})).Should(Succeed())

				out, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(Equal(ctrl.Result{}))

				serverClaim := &metalv1alpha1.ServerClaim{}
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(metalMachine), serverClaim)

				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue(), "ServerClaim should not exist because of early return")
			})
		})
		When("bootstrap data is empty", func() {
			It("should return empty and not create resources", func() {
				Eventually(Update(machine, func() {
					machine.Spec.Bootstrap.DataSecretName = nil
					machine.Spec.Bootstrap.ConfigRef = clusterapiv1beta2.ContractVersionedObjectReference{
						APIGroup: "bootstrap.cluster.x-k8s.io",
						Kind:     "dummy-kind",
						Name:     "dummy-config",
					}
				})).Should(Succeed())
				out, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(metalMachine),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(Equal(ctrl.Result{}))
			})
		})
	})
})
