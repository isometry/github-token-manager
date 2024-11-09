package v1

import "k8s.io/apimachinery/pkg/types"

type secretOwner interface {
	GetSecretNamespace() string
	GetSecretName() string
	GetSecretBasicAuth() bool
}

type ManagedSecret struct {
	BasicAuth bool   `json:"basicAuth"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

func (m ManagedSecret) IsUnset() bool {
	return m.Name == ""
}

func (m ManagedSecret) MatchesSpec(owner secretOwner) bool {
	return m.Namespace == owner.GetSecretNamespace() && m.Name == owner.GetSecretName() && m.BasicAuth == owner.GetSecretBasicAuth()
}

// func (m *ManagedSecret) MatchesKey(key types.NamespacedName) bool {
// 	return m.Namespace == key.Namespace && m.Name == key.Name
// }

func (m ManagedSecret) Key() types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.Namespace,
		Name:      m.Name,
	}
}
