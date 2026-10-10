package seeder

import (
	"fmt"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

type _PortalEntryHTTP struct {
	HttpEnabled  *bool `yaml:"httpEnabled"`
	HttpPort     *int  `yaml:"httpPort"`
	HttpsEnabled *bool `yaml:"httpsEnabled"`
	HttpsPort    *int  `yaml:"httpsPort"`
	AutoHTTPS    *bool `yaml:"autoHTTPS"`
}

func (h *_PortalEntryHTTP) UnmarshalYAML(node *yaml.Node) error {
	fields, err := seedMappingFields(node, "portal entry http")
	if err != nil {
		return err
	}
	accepted := yamlFieldsOf(_PortalEntryHTTP{})
	for key, value := range fields {
		if !accepted[key] {
			return fmt.Errorf("portal entry http: unknown field %q", key)
		}
		if value.Tag == "!!null" {
			return fmt.Errorf("portal entry http: %s cannot be null", key)
		}
	}
	type plain _PortalEntryHTTP
	return node.Decode((*plain)(h))
}

func (h _PortalEntryHTTP) update() core.PortalEntryHTTPUpdate {
	return core.PortalEntryHTTPUpdate{HttpEnabled: h.HttpEnabled, HttpPort: h.HttpPort, HttpsEnabled: h.HttpsEnabled, HttpsPort: h.HttpsPort, AutoHTTPS: h.AutoHTTPS}
}
