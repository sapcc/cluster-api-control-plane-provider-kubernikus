/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubernikusControlPlaneSpec defines the desired state of KubernikusControlPlane
type KubernikusControlPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Version string `json:"version"`

	ServiceCidr string `json:"serviceCidr,omitempty"`

	AdvertiseAddress string `json:"advertiseAddress,omitempty"`
	AdvertisePort    int64  `json:"advertisePort,omitempty"`

	Backup string `json:"backup,omitempty"`

	CustomCNI bool `json:"customCNI,omitempty"`

	DnsAddress string `json:"dnsAddress,omitempty"`
	DnsDomain  string `json:"dnsDomain,omitempty"`

	SeedKubeadm bool `json:"seedKubeadm,omitempty"`

	SSHPublicKey string `json:"sshPublicKey,omitempty"`

	Oidc *OIDC `json:"oidc,omitempty"`
}

// KubernikusControlPlaneStatus defines the observed state of KubernikusControlPlane
type KubernikusControlPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Initialized bool `json:"initialized"`
	Ready       bool `json:"ready"`

	FailureReason  string `json:"failureReason,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`

	Version    string             `json:"version"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ExternalManagedControlPlane indicates to Cluster API that the Control Plane
	// is externally managed by Kubernikus.
	// +kubebuilder:default=true
	ExternalManagedControlPlane *bool `json:"externalManagedControlPlane"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KubernikusControlPlane is the Schema for the kubernikuscontrolplanes API
type KubernikusControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernikusControlPlaneSpec   `json:"spec,omitempty"`
	Status KubernikusControlPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KubernikusControlPlaneList contains a list of KubernikusControlPlane
type KubernikusControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernikusControlPlane `json:"items"`
}

type OIDC struct {
	// client ID
	ClientID string `json:"clientID,omitempty"`
	// issuer URL
	IssuerURL string `json:"issuerURL,omitempty"`
}

func init() {
	SchemeBuilder.Register(&KubernikusControlPlane{}, &KubernikusControlPlaneList{})
}
