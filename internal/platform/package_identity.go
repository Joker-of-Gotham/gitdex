package platform

import "strings"

type PackageIdentity struct {
	Registry    string `json:"registry,omitempty"`
	PackageType string `json:"package_type,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

func NormalizePackageIdentity(identity PackageIdentity) PackageIdentity {
	return PackageIdentity{
		Registry:    strings.TrimSpace(identity.Registry),
		PackageType: strings.TrimSpace(identity.PackageType),
		Namespace:   strings.TrimSpace(identity.Namespace),
		Name:        strings.TrimSpace(identity.Name),
		Version:     strings.TrimSpace(identity.Version),
		Scope:       strings.TrimSpace(identity.Scope),
	}
}
