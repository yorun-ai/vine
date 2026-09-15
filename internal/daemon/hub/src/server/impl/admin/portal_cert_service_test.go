package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestPortalCertServiceMapsListItemsAndDetail(t *testing.T) {
	repo := &_MaintenanceServicePortalCertRepo{items: map[string]*core.PortalCert{
		"demo-cert": {
			Id:               5,
			Name:             "demo-cert",
			Issuer:           "letsencrypt",
			Domains:          []string{"demo.local"},
			PublicKeyBase64:  "pub",
			PrivateKeyBase64: "pri",
			FieldSources:     core.FieldSources{"/publicKeyBase64": {Source: "app/default", Override: "hub"}},
		},
	}}
	service := &PortalCertApiServiceServerImpl{PortalCertCore: &core.PortalCertCore{PortalCertRepo: repo}}

	list := service.List()

	require.Len(t, list, 1)
	assert.Equal(t, 5, list[0].Id)
	assert.Equal(t, "demo-cert", list[0].Name)
	assert.Equal(t, []string{"demo.local"}, list[0].Domains)
	assert.True(t, list[0].PrivateKeyConfigured)

	detail := service.Get(5)

	require.Len(t, detail.FieldSources, 1)
	assert.Equal(t, "/publicKeyBase64", detail.FieldSources[0].Path)
	assert.Equal(t, "app/default", detail.FieldSources[0].Source)
}
