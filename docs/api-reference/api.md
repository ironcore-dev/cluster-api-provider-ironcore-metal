<p>Packages:</p>
<ul>
<li>
<a href="#infrastructure.cluster.x-k8s.io%2fv1alpha1">infrastructure.cluster.x-k8s.io/v1alpha1</a>
</li>
</ul>
<h2 id="infrastructure.cluster.x-k8s.io/v1alpha1">infrastructure.cluster.x-k8s.io/v1alpha1</h2>
<div>
<p>Package v1alpha1 contains API Schema definitions for the settings.gardener.cloud API group</p>
</div>
Resource Types:
<ul></ul>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IPAMConfig">IPAMConfig
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineSpec">IroncoreMetalMachineSpec</a>)
</p>
<div>
<p>IPAMConfig is a reference to an IPAM resource.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadataKey</code><br/>
<em>
string
</em>
</td>
<td>
<p>MetadataKey is the name of metadata key for the network.</p>
</td>
</tr>
<tr>
<td>
<code>ipamRef</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IPAMObjectReference">
IPAMObjectReference
</a>
</em>
</td>
<td>
<p>IPAMRef is a reference to the IPAM object, which will be used for IP allocation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IPAMObjectReference">IPAMObjectReference
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IPAMConfig">IPAMConfig</a>)
</p>
<div>
<p>IPAMObjectReference is a reference to the IPAM object, which will be used for IP allocation.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of resource being referenced.</p>
</td>
</tr>
<tr>
<td>
<code>apiGroup</code><br/>
<em>
string
</em>
</td>
<td>
<p>APIGroup is the group for the resource being referenced.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
string
</em>
</td>
<td>
<p>Kind is the type of resource being referenced.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalCluster">IroncoreMetalCluster
</h3>
<div>
<p>IroncoreMetalCluster is the Schema for the ironcoremetalclusters API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterSpec">
IroncoreMetalClusterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta2.APIEndpoint
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.</p>
</td>
</tr>
<tr>
<td>
<code>clusterNetwork</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta2.ClusterNetwork
</em>
</td>
<td>
<em>(Optional)</em>
<p>Cluster network configuration.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterStatus">
IroncoreMetalClusterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterInitializationStatus">IroncoreMetalClusterInitializationStatus
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterStatus">IroncoreMetalClusterStatus</a>)
</p>
<div>
<p>IroncoreMetalClusterInitializationStatus provides observations of the IroncoreMetalCluster initialization process.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provisioned</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Provisioned is true when the infrastructure provider reports that the Cluster&rsquo;s infrastructure is fully provisioned.
NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Cluster provisioning.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterSpec">IroncoreMetalClusterSpec
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalCluster">IroncoreMetalCluster</a>)
</p>
<div>
<p>IroncoreMetalClusterSpec defines the desired state of IroncoreMetalCluster</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>controlPlaneEndpoint</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta2.APIEndpoint
</em>
</td>
<td>
<em>(Optional)</em>
<p>ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.</p>
</td>
</tr>
<tr>
<td>
<code>clusterNetwork</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta2.ClusterNetwork
</em>
</td>
<td>
<em>(Optional)</em>
<p>Cluster network configuration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterStatus">IroncoreMetalClusterStatus
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalCluster">IroncoreMetalCluster</a>)
</p>
<div>
<p>IroncoreMetalClusterStatus defines the observed state of IroncoreMetalCluster</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ready</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ready denotes that the cluster (infrastructure) is ready.
Deprecated: This field is part of the v1beta1 contract and will be ignored in the future.</p>
</td>
</tr>
<tr>
<td>
<code>initialization,omitempty,omitzero</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalClusterInitializationStatus">
IroncoreMetalClusterInitializationStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Initialization provides observations of the IroncoreMetalCluster initialization process.
NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Cluster provisioning.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions defines current service state of the IroncoreMetalCluster.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachine">IroncoreMetalMachine
</h3>
<div>
<p>IroncoreMetalMachine is the Schema for the ironcoremetalmachines API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineSpec">
IroncoreMetalMachineSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>Image specifies the boot image to be used for the server.</p>
</td>
</tr>
<tr>
<td>
<code>serverSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerSelector specifies matching criteria for labels on Servers.
This is used to claim specific Server types for a IroncoreMetalMachine.</p>
</td>
</tr>
<tr>
<td>
<code>ipamConfig</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IPAMConfig">
[]IPAMConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPAMConfig is a list of references to Network resources that should be used to assign IP addresses to the worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</em>
</td>
<td>
<em>(Optional)</em>
<p>Metadata is a key-value map of additional data which should be passed to the Machine.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineStatus">
IroncoreMetalMachineStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineInitializationStatus">IroncoreMetalMachineInitializationStatus
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineStatus">IroncoreMetalMachineStatus</a>)
</p>
<div>
<p>IroncoreMetalMachineInitializationStatus provides observations of the IroncoreMetalMachine initialization process.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provisioned</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Provisioned is true when the infrastructure provider reports that the Machine&rsquo;s infrastructure is fully provisioned.
NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineSpec">IroncoreMetalMachineSpec
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachine">IroncoreMetalMachine</a>, <a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateResource">IroncoreMetalMachineTemplateResource</a>)
</p>
<div>
<p>IroncoreMetalMachineSpec defines the desired state of IroncoreMetalMachine</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>Image specifies the boot image to be used for the server.</p>
</td>
</tr>
<tr>
<td>
<code>serverSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerSelector specifies matching criteria for labels on Servers.
This is used to claim specific Server types for a IroncoreMetalMachine.</p>
</td>
</tr>
<tr>
<td>
<code>ipamConfig</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IPAMConfig">
[]IPAMConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPAMConfig is a list of references to Network resources that should be used to assign IP addresses to the worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</em>
</td>
<td>
<em>(Optional)</em>
<p>Metadata is a key-value map of additional data which should be passed to the Machine.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineStatus">IroncoreMetalMachineStatus
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachine">IroncoreMetalMachine</a>)
</p>
<div>
<p>IroncoreMetalMachineStatus defines the observed state of IroncoreMetalMachine</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ready</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ready indicates the Machine infrastructure has been provisioned and is ready.
Deprecated: This field is part of the v1beta1 contract and will be removed in the future.</p>
</td>
</tr>
<tr>
<td>
<code>initialization,omitempty,omitzero</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineInitializationStatus">
IroncoreMetalMachineInitializationStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Initialization provides observations of the IroncoreMetalMachine initialization process.
NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Machine provisioning.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions defines current service state of the IroncoreMetalMachine</p>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplate">IroncoreMetalMachineTemplate
</h3>
<div>
<p>IroncoreMetalMachineTemplate is the Schema for the ironcoremetalmachinetemplates API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateSpec">
IroncoreMetalMachineTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateResource">
IroncoreMetalMachineTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateResource">IroncoreMetalMachineTemplateResource
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateSpec">IroncoreMetalMachineTemplateSpec</a>)
</p>
<div>
<p>IroncoreMetalMachineTemplateResource defines the spec and metadata for IroncoreMetalMachineTemplate supported by capi.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
sigs.k8s.io/cluster-api/api/core/v1beta2.ObjectMeta
</em>
</td>
<td>
<em>(Optional)</em>
<p>Standard object&rsquo;s metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineSpec">
IroncoreMetalMachineSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>providerID</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProviderID is the unique identifier as specified by the cloud provider.</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br/>
<em>
string
</em>
</td>
<td>
<p>Image specifies the boot image to be used for the server.</p>
</td>
</tr>
<tr>
<td>
<code>serverSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerSelector specifies matching criteria for labels on Servers.
This is used to claim specific Server types for a IroncoreMetalMachine.</p>
</td>
</tr>
<tr>
<td>
<code>ipamConfig</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IPAMConfig">
[]IPAMConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPAMConfig is a list of references to Network resources that should be used to assign IP addresses to the worker nodes.</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</em>
</td>
<td>
<em>(Optional)</em>
<p>Metadata is a key-value map of additional data which should be passed to the Machine.</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateSpec">IroncoreMetalMachineTemplateSpec
</h3>
<p>
(<em>Appears on:</em><a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplate">IroncoreMetalMachineTemplate</a>)
</p>
<div>
<p>IroncoreMetalMachineTemplateSpec defines the desired state of IroncoreMetalMachineTemplate</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#infrastructure.cluster.x-k8s.io/v1alpha1.IroncoreMetalMachineTemplateResource">
IroncoreMetalMachineTemplateResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
