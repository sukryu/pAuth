package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// User defines the user resource
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

type UserSpec struct {
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"passwordHash,omitempty"`
	Roles        []string `json:"roles,omitempty"`
}

type UserStatus struct {
	Active    bool         `json:"active"`
	LastLogin *metav1.Time `json:"lastLogin,omitempty"`
}

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// DeepCopy implements runtime.Object interface
func (in *User) DeepCopy() *User {
	if in == nil {
		return nil
	}
	out := new(User)
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	if in.Spec.Roles != nil {
		out.Spec.Roles = make([]string, len(in.Spec.Roles))
		copy(out.Spec.Roles, in.Spec.Roles)
	}
	return out
}

// DeepCopyObject implements runtime.Object interface
func (in *User) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// DeepCopyInto copies all properties into another User
func (in *User) DeepCopyInto(out *User) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	if in.Spec.Roles != nil {
		out.Spec.Roles = make([]string, len(in.Spec.Roles))
		copy(out.Spec.Roles, in.Spec.Roles)
	}
}

// DeepCopy for UserList using DeepCopyInto
func (in *UserList) DeepCopy() *UserList {
	if in == nil {
		return nil
	}
	out := new(UserList)
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]User, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
	return out
}
