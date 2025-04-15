// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/base64"
	"fmt"

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
	clusterapiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiv1beta1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"

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
			cluster              *clusterapiv1beta1.Cluster
			machine              *clusterapiv1beta1.Machine
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
			}

			cluster = &clusterapiv1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster",
					Namespace: namespace,
				},
				Spec: clusterapiv1beta1.ClusterSpec{
					InfrastructureRef: &corev1.ObjectReference{
						APIVersion: infrav1alpha1.GroupVersion.String(),
						Kind:       "IroncoreMetalCluster",
						Name:       metalCluster.Name,
					},
				},
			}

			machine = &clusterapiv1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine",
					Namespace: namespace,
					Labels:    map[string]string{clusterapiv1beta1.ClusterNameLabel: cluster.Name},
				},
				Spec: clusterapiv1beta1.MachineSpec{
					ClusterName: cluster.Name,
					Bootstrap: clusterapiv1beta1.Bootstrap{
						DataSecretName: &secret.Name,
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
				cluster.Status.InfrastructureReady = true
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

		When("the ipam config is present in the metal machine", func() {
			const metadataKey = "meta-key"

			var (
				ipAddressClaim *capiv1beta1.IPAddressClaim
				ipAddress      *capiv1beta1.IPAddress
			)

			BeforeEach(func() {
				ipAddressClaim = &capiv1beta1.IPAddressClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", metalMachine.Name, metadataKey),
						Namespace: namespace,
					},
				}

				ipAddress = &capiv1beta1.IPAddress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ip-address",
						Namespace: namespace,
					},
					Spec: capiv1beta1.IPAddressSpec{
						Address: "10.11.12.13",
						Prefix:  24,
						Gateway: "10.11.12.1",
					},
				}

				metalMachine.Spec.IPAMConfig = []infrav1alpha1.IPAMConfig{{
					MetadataKey: metadataKey,
					IPAMRef: &infrav1alpha1.IPAMObjectReference{
						Name:     "pool",
						APIGroup: "ipam.cluster.x-k8s.io/v1alpha2",
						Kind:     "GlobalInClusterIPPool",
					}}}

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
		})
	})
})
