package skeled

import "fmt"

// PortalCorsMode Portal site CORS mode
type PortalCorsMode string

const (
	PortalCorsModeUnspecified PortalCorsMode = "UNSPECIFIED"
	// PortalCorsModeDisabled Disable CORS
	PortalCorsModeDisabled PortalCorsMode = "DISABLED"
	// PortalCorsModeSameDomain Allow origins in the same domain as the entry rule
	PortalCorsModeSameDomain PortalCorsMode = "SAME_DOMAIN"
	// PortalCorsModeStrict Only allow Origins in the configuration list
	PortalCorsModeStrict PortalCorsMode = "STRICT"
)

func (portalCorsMode PortalCorsMode) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", portalCorsMode)), nil
}

func (portalCorsMode *PortalCorsMode) UnmarshalJSON(data []byte) error {
	var err error = nil
	switch string(data) {
	case `"DISABLED"`:
		*portalCorsMode = PortalCorsModeDisabled
	case `"SAME_DOMAIN"`:
		*portalCorsMode = PortalCorsModeSameDomain
	case `"STRICT"`:
		*portalCorsMode = PortalCorsModeStrict
	case "null":
		err = fmt.Errorf("unexpected null value for non-pointer PortalCorsMode")
	default:
		*portalCorsMode = PortalCorsModeUnspecified
	}
	return err
}

// PortalSiteType Portal target site type
type PortalSiteType string

const (
	PortalSiteTypeUnspecified PortalSiteType = "UNSPECIFIED"
	// PortalSiteTypeRpcgw Rpc gateway
	PortalSiteTypeRpcgw PortalSiteType = "RPCGW"
	// PortalSiteTypeWebgw Web gateway
	PortalSiteTypeWebgw PortalSiteType = "WEBGW"
)

func (portalSiteType PortalSiteType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", portalSiteType)), nil
}

func (portalSiteType *PortalSiteType) UnmarshalJSON(data []byte) error {
	var err error = nil
	switch string(data) {
	case `"RPCGW"`:
		*portalSiteType = PortalSiteTypeRpcgw
	case `"WEBGW"`:
		*portalSiteType = PortalSiteTypeWebgw
	case "null":
		err = fmt.Errorf("unexpected null value for non-pointer PortalSiteType")
	default:
		*portalSiteType = PortalSiteTypeUnspecified
	}
	return err
}
