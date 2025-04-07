/*
Copyright 2025 Flant JSC

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

package integration

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

type StaticInstanceStatusCurrentStatusPhase string

const (
	StaticInstanceStatusCurrentStatusPhaseError         StaticInstanceStatusCurrentStatusPhase = "Error"
	StaticInstanceStatusCurrentStatusPhasePending       StaticInstanceStatusCurrentStatusPhase = "Pending"
	StaticInstanceStatusCurrentStatusPhaseBootstrapping StaticInstanceStatusCurrentStatusPhase = "Bootstrapping"
	StaticInstanceStatusCurrentStatusPhaseRunning       StaticInstanceStatusCurrentStatusPhase = "Running"
	StaticInstanceStatusCurrentStatusPhaseCleaning      StaticInstanceStatusCurrentStatusPhase = "Cleaning"
)

var (
	GroupVersion    = schema.GroupVersion{Group: "deckhouse.io", Version: "v1alpha1"}
	DhSchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
)

func init() {
	DhSchemeBuilder.Register(&SSHCredentials{}, &SSHCredentialsList{})
	DhSchemeBuilder.Register(&StaticInstance{}, &StaticInstanceList{})
}

/*  SSHCredentials  */

type SSHCredentialsSpec struct {
	User          string `json:"user"`
	PrivateSSHKey string `json:"privateSSHKey"`
	SudoPassword  string `json:"sudoPassword,omitempty"`

	//+kubebuilder:default:=22
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	SSHPort int `json:"sshPort,omitempty"`

	SSHExtraArgs string `json:"sshExtraArgs,omitempty"`
}

type SSHCredentials struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHCredentialsSpec `json:"spec,omitempty"`
}

type SSHCredentialsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSHCredentials `json:"items"`
}

func GetSSHCredentialsRef(name string) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		APIVersion: "deckhouse.io/v1alpha1",
		Kind:       "SSHCredentials",
		Name:       name,
	}
}

func (in *SSHCredentialsSpec) DeepCopyInto(out *SSHCredentialsSpec) {
	*out = *in
}

func (in *SSHCredentialsSpec) DeepCopy() *SSHCredentialsSpec {
	if in == nil {
		return nil
	}
	out := new(SSHCredentialsSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *SSHCredentials) DeepCopyInto(out *SSHCredentials) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *SSHCredentials) DeepCopy() *SSHCredentials {
	if in == nil {
		return nil
	}
	out := new(SSHCredentials)
	in.DeepCopyInto(out)
	return out
}

func (in *SSHCredentials) DeepCopyObject() apiruntime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *SSHCredentialsList) DeepCopyInto(out *SSHCredentialsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SSHCredentials, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *SSHCredentialsList) DeepCopy() *SSHCredentialsList {
	if in == nil {
		return nil
	}
	out := new(SSHCredentialsList)
	in.DeepCopyInto(out)
	return out
}

func (in *SSHCredentialsList) DeepCopyObject() apiruntime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

/*  Static Instance  */

type StaticInstanceSpec struct {
	Address        string                  `json:"address"`
	CredentialsRef *corev1.ObjectReference `json:"credentialsRef"`
}

// StaticInstanceStatus defines the observed state of StaticInstance
type StaticInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`

	// +optional
	NodeRef *corev1.ObjectReference `json:"nodeRef,omitempty"`

	// +optional
	CurrentStatus *StaticInstanceStatusCurrentStatus `json:"currentStatus,omitempty"`

	// Conditions defines current service state of the StaticInstance.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

type StaticInstanceStatusCurrentStatus struct {
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`

	// +optional
	// +kubebuilder:validation:Enum=Error;Pending;Bootstrapping;Running;Cleaning
	Phase StaticInstanceStatusCurrentStatusPhase `json:"phase"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.currentStatus.phase",description="Static instance state"
//+kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.nodeRef.name",description="Node associated with this static instance"
//+kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".status.machineRef.name",description="Static machine associated with this static instance"

// StaticInstance is the Schema for the staticinstances API
type StaticInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StaticInstanceSpec   `json:"spec,omitempty"`
	Status StaticInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// StaticInstanceList contains a list of StaticInstance
type StaticInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StaticInstance `json:"items"`
}

func (in *StaticInstance) GetConditions() clusterv1.Conditions {
	return in.Status.Conditions
}

func (in *StaticInstance) SetConditions(conditions clusterv1.Conditions) {
	in.Status.Conditions = conditions
}

func (in *StaticInstance) DeepCopyInto(out *StaticInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *StaticInstance) DeepCopy() *StaticInstance {
	if in == nil {
		return nil
	}
	out := new(StaticInstance)
	in.DeepCopyInto(out)
	return out
}

func (in *StaticInstance) DeepCopyObject() apiruntime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *StaticInstanceList) DeepCopyInto(out *StaticInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StaticInstance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *StaticInstanceList) DeepCopy() *StaticInstanceList {
	if in == nil {
		return nil
	}
	out := new(StaticInstanceList)
	in.DeepCopyInto(out)
	return out
}

func (in *StaticInstanceList) DeepCopyObject() apiruntime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *StaticInstanceSpec) DeepCopyInto(out *StaticInstanceSpec) {
	*out = *in
	if in.CredentialsRef != nil {
		in, out := &in.CredentialsRef, &out.CredentialsRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
}

func (in *StaticInstanceSpec) DeepCopy() *StaticInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(StaticInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *StaticInstanceStatus) DeepCopyInto(out *StaticInstanceStatus) {
	*out = *in
	if in.MachineRef != nil {
		in, out := &in.MachineRef, &out.MachineRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	if in.NodeRef != nil {
		in, out := &in.NodeRef, &out.NodeRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	if in.CurrentStatus != nil {
		in, out := &in.CurrentStatus, &out.CurrentStatus
		*out = new(StaticInstanceStatusCurrentStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(clusterv1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *StaticInstanceStatus) DeepCopy() *StaticInstanceStatus {
	if in == nil {
		return nil
	}
	out := new(StaticInstanceStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *StaticInstanceStatusCurrentStatus) DeepCopyInto(out *StaticInstanceStatusCurrentStatus) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
}

func (in *StaticInstanceStatusCurrentStatus) DeepCopy() *StaticInstanceStatusCurrentStatus {
	if in == nil {
		return nil
	}
	out := new(StaticInstanceStatusCurrentStatus)
	in.DeepCopyInto(out)
	return out
}
