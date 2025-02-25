package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +openshift:compatibility-gen:level=4

// ConsolePlugin is an extension for customizing OpenShift web console by
// dynamically loading code from another service running on the cluster.
//
// Compatibility level 4: No compatibility is provided, the API can change at any point for any reason. These capabilities should not be used by applications needing long term support.
type ConsolePlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +kubebuilder:validation:Required
	// +required
	Spec ConsolePluginSpec `json:"spec"`
}

// ConsolePluginSpec is the desired plugin configuration.
type ConsolePluginSpec struct {
	// displayName is the display name of the plugin.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +optional
	DisplayName string `json:"displayName"`
	// service is a Kubernetes Service that exposes the plugin using a
	// deployment with an HTTP server. The Service must use HTTPS and
	// Service serving certificate. The console backend will proxy the
	// plugins assets from the Service using the service CA bundle.
	// +kubebuilder:validation:Required
	// +required
	Service ConsolePluginService `json:"service"`
	// proxy is a list of Services that the plugin needs to connect to.
	// +kubebuilder:validation:Optional
	// +optional
	Proxy ConsolePluginProxy `json:"proxy"`
}

// ConsolePluginProxy holds information on various service types
// to which console's backend will proxy the plugin's requests.
type ConsolePluginProxy struct {
	// services is a list of in-cluster Services that the plugin
	// will connect to. The Service must use HTTPS. Console backend
	// exposes the following endpoint in order to proxy communication
	// between the plugin and the Service:
	//
	// /api/proxy/namespace/<service-namespace>/service/<service-name>:<port-number>/<request-path>?<optional-query-parameters>
	//
	// Request example path:
	//
	// /api/proxy/namespace/helm/service/helm-charts:8443/releases?limit=10
	//
	// +kubebuilder:validation:Optional
	// +optional
	Services []ConsolePluginProxyService `json:"services"`
}

// ConsolePluginProxyService holds information on Service to which
// console's backend will proxy the plugin's requests.
type ConsolePluginProxyService struct {
	// name of Service that the plugin needs to connect to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +required
	Name string `json:"name"`
	// namespace of Service that the plugin needs to connect to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +required
	Namespace string `json:"namespace"`
	// port on which the Service that the plugin needs to connect to
	// is listening on.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Maximum:=65535
	// +kubebuilder:validation:Minimum:=1
	// +required
	Port int32 `json:"port"`
	// caCertificate provides the cert authority certificate contents,
	// in case the proxied Service is using custom service CA.
	// By default service CA bundle is used.
	// +kubebuilder:validation:Pattern=`^-----BEGIN CERTIFICATE-----([\s\S]*)-----END CERTIFICATE-----\s?$`
	// +kubebuilder:validation:Optional
	// +optional
	CACertificate string `json:"caCertificate,omitempty"`
	// authorize indicates if the proxied request will logged-in user's
	// OpenShift access token in the "Authorization" request header:
	//
	// Authorization: Bearer sha256~kV46hPnEYhCWFnB85r5NrprAxggzgb6GOeLbgcKNsH0
	//
	// By default the access token is not part of the proxied request.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +optional
	Authorize bool `json:"authorize,omitempty"`
}

// ConsolePluginService holds information on Service that is serving
// console dynamic plugin assets.
type ConsolePluginService struct {
	// name of Service that is serving the plugin assets.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +required
	Name string `json:"name"`
	// namespace of Service that is serving the plugin assets.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +required
	Namespace string `json:"namespace"`
	// port on which the Service that is serving the plugin is listening to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Maximum:=65535
	// +kubebuilder:validation:Minimum:=1
	// +required
	Port int32 `json:"port"`
	// basePath is the path to the plugin's assets. The primary asset it the
	// manifest file called `plugin-manifest.json`, which is a JSON document
	// that contains metadata about the plugin and the extensions.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^/`
	// +kubebuilder:default:="/"
	// +required
	BasePath string `json:"basePath"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +openshift:compatibility-gen:level=4

// Compatibility level 4: No compatibility is provided, the API can change at any point for any reason. These capabilities should not be used by applications needing long term support.
type ConsolePluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ConsolePlugin `json:"items"`
}
