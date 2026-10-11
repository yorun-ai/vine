package seeder

import (
	"fmt"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"gopkg.in/yaml.v3"
)

type _PortalEntryHTTP struct {
	Host      *string               `yaml:"host"`
	Http      core.PortalSwitchMode `yaml:"http"`
	HttpPort  *int                  `yaml:"httpPort"`
	Https     core.PortalSwitchMode `yaml:"https"`
	HttpsPort *int                  `yaml:"httpsPort"`
	AutoHTTPS core.PortalSwitchMode `yaml:"autohttps"`
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
		if key == "http" || key == "https" || key == "autohttps" {
			if value.Tag != "!!str" || !core.PortalSwitchMode(value.Value).Valid() {
				return fmt.Errorf("portal entry http.%s must be ENABLED, DISABLED or AUTO", key)
			}
		}
	}
	type plain _PortalEntryHTTP
	return node.Decode((*plain)(h))
}

func (h _PortalEntryHTTP) config() core.PortalEntryHTTP {
	value := core.PortalEntryHTTP{Http: h.Http, Https: h.Https, AutoHTTPS: h.AutoHTTPS}
	if h.Host != nil {
		value.Host = *h.Host
	}
	if h.HttpPort != nil {
		value.HttpPort = *h.HttpPort
	}
	if h.HttpsPort != nil {
		value.HttpsPort = *h.HttpsPort
	}
	return value
}
