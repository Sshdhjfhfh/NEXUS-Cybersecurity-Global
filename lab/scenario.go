package lab

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Kind string

const (
	DVWA            Kind = "dvwa"
	Metasploitable  Kind = "metasploitable"
	VulnHub         Kind = "vulnhub"
	ActiveDirectory Kind = "active_directory"
)

type Scenario struct {
	Kind Kind
	// The operator supplies locally available, digest-pinned images. No images
	// are downloaded and no vulnerable services are started by this library.
	Images map[string]string
}

type Profile struct {
	Kind  Kind
	Roles []string
}

func Profiles() []Profile {
	return []Profile{{DVWA, []string{"web", "database"}}, {Metasploitable, []string{"target"}},
		{VulnHub, []string{"target"}}, {ActiveDirectory, []string{"controller", "member"}}}
}

var pinnedImage = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:[a-f0-9]{64}$`)

// Compose produces a document, not a shell command. Networks are internal,
// ports are unpublished, host networking and privileged containers are absent,
// and the default seccomp profile remains enabled. VM-only appliances must be
// supplied through an explicit container adapter; this is not a VM importer.
func (s Scenario) Compose() ([]byte, error) {
	var roles []string
	for _, p := range Profiles() {
		if p.Kind == s.Kind {
			roles = p.Roles
			break
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("unsupported laboratory kind %q", s.Kind)
	}
	if len(s.Images) != len(roles) {
		return nil, fmt.Errorf("scenario requires exactly %d images", len(roles))
	}
	services := map[string]any{}
	for _, role := range roles {
		image := s.Images[role]
		if !pinnedImage.MatchString(image) || strings.Contains(image, "..") {
			return nil, fmt.Errorf("role %s requires a digest-pinned image", role)
		}
		services[role] = map[string]any{
			"image": image, "pull_policy": "never", "restart": "no", "networks": []string{"lab"},
			"security_opt": []string{"no-new-privileges:true"}, "cap_drop": []string{"ALL"},
			"pids_limit": 256, "mem_limit": "1g", "cpus": 1.0,
			"labels": map[string]string{"nexus.lab": "true", "nexus.lab.kind": string(s.Kind)},
		}
	}
	return yaml.Marshal(map[string]any{"name": "nexus-lab-" + string(s.Kind), "services": services,
		"networks": map[string]any{"lab": map[string]any{"internal": true, "driver": "bridge"}}})
}
