package admin

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestPortalCertServiceMapsListItemsAndDetail(t *testing.T) {
	repo := &_PortalCertRepoSpy{items: map[string]*core.PortalCert{
		"demo-cert": {
			Id:           5,
			Name:         "demo-cert",
			Issuer:       "letsencrypt",
			Domains:      []string{"demo.local"},
			Certificate:  "pub",
			PrivateKey:   "pri",
			FieldSources: core.FieldSources{"/certificate": {Source: "app/default", Override: "hub"}},
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
	assert.Equal(t, "/certificate", detail.FieldSources[0].Path)
	assert.Equal(t, "app/default", detail.FieldSources[0].Source)
}

func TestPortalCertServiceDoesNotReturnPrivateKeyProvenanceValues(t *testing.T) {
	secret := skel.JSON(`"private PEM content"`)
	sources := core.FieldSources{
		"/privateKey": {Source: "seed", Template: &secret, Bindings: []core.FieldSourceBinding{{Variable: "key", Value: secret}}},
	}
	fields := portalCertFieldSources(sources)
	require.Len(t, fields, 1)
	assert.Equal(t, "seed", fields[0].Source)
	assert.Nil(t, fields[0].Template)
	assert.Empty(t, fields[0].Bindings)
	assert.NotContains(t, vcode.MustMarshalJsonS(fields), "private PEM content")
	assert.NotNil(t, sources["/privateKey"].Template, "response redaction must not mutate stored provenance")
}
