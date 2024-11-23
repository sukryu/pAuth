package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Role 정의
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Rules []PolicyRule `json:"rules"`
}

// PolicyRule 정의
type PolicyRule struct {
	Verbs     []string `json:"verbs"`     // create, read, update, delete
	Resources []string `json:"resources"` // users, roles, etc.
	APIGroups []string `json:"apiGroups"` // auth.service
}

// RoleBinding 정의
type RoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Subjects []Subject `json:"subjects"`
	RoleRef  RoleRef   `json:"roleRef"`
}

type Subject struct {
	Kind string `json:"kind"` // User, Group
	Name string `json:"name"`
}

type RoleRef struct {
	Kind string `json:"kind"` // Role
	Name string `json:"name"`
}

// DeepCopyInto copies the receiver into out
func (in *Role) DeepCopyInto(out *Role) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	if in.Rules != nil {
		out.Rules = make([]PolicyRule, len(in.Rules))
		for i := range in.Rules {
			in.Rules[i].DeepCopyInto(&out.Rules[i])
		}
	}
}

// DeepCopy creates a deep copy of Role
func (in *Role) DeepCopy() *Role {
	if in == nil {
		return nil
	}
	out := new(Role)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies PolicyRule into out
func (in *PolicyRule) DeepCopyInto(out *PolicyRule) {
	*out = *in
	if in.Verbs != nil {
		out.Verbs = make([]string, len(in.Verbs))
		copy(out.Verbs, in.Verbs)
	}
	if in.Resources != nil {
		out.Resources = make([]string, len(in.Resources))
		copy(out.Resources, in.Resources)
	}
	if in.APIGroups != nil {
		out.APIGroups = make([]string, len(in.APIGroups))
		copy(out.APIGroups, in.APIGroups)
	}
}

// DeepCopyInto copies RoleBinding into out
func (in *RoleBinding) DeepCopyInto(out *RoleBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.RoleRef = in.RoleRef

	if in.Subjects != nil {
		out.Subjects = make([]Subject, len(in.Subjects))
		for i := range in.Subjects {
			in.Subjects[i].DeepCopyInto(&out.Subjects[i])
		}
	}
}

// DeepCopy creates a deep copy of RoleBinding
func (in *RoleBinding) DeepCopy() *RoleBinding {
	if in == nil {
		return nil
	}
	out := new(RoleBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies Subject into out
func (in *Subject) DeepCopyInto(out *Subject) {
	*out = *in
}
