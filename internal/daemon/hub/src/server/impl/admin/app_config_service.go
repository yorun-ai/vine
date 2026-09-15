package admin

import (
	"cmp"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type AppConfigApiServiceServerImpl struct {
	skeled.DefaultAppConfigApiServiceServer

	AppConfigCore *core.AppConfigCore `inject:""`
}

func (s *AppConfigApiServiceServerImpl) List() []skeled.AppConfigListItem {
	slots := s.AppConfigCore.List()
	items := make([]skeled.AppConfigListItem, 0, len(slots))
	for _, slot := range sortedAppConfigSlots(slots) {
		items = append(items, toServerAppConfigListItem(slot))
	}
	return items
}

func (s *AppConfigApiServiceServerImpl) Get(key string) skeled.AppConfigItem {
	return s.toServerAppConfigItem(s.AppConfigCore.GetSlot(key))
}

func (s *AppConfigApiServiceServerImpl) Update(id int, update skeled.AppConfigUpdate) skeled.AppConfigItem {
	return s.toServerAppConfigItem(s.AppConfigCore.Update(id, core.AppConfigUpdate{
		Value: update.Value,
	}))
}

func (s *AppConfigApiServiceServerImpl) Create(creation skeled.AppConfigCreation) skeled.AppConfigItem {
	ex.PanicNewIfNot(isValidConfigSkelName(creation.SkelName), ex.OperationFailed, ex.F("invalid config skelName %q", creation.SkelName))
	return s.toServerAppConfigItem(s.AppConfigCore.Create(core.AppConfigCreation{
		Name:  creation.SkelName,
		Value: creation.Value,
	}))
}

func (s *AppConfigApiServiceServerImpl) Remove(id int) bool {
	item := s.AppConfigCore.Get(id)
	ex.PanicNewIfNot(item.Definition == nil, ex.OperationFailed, ex.F("config %q is not unused", item.Name))
	return s.AppConfigCore.Remove(id)
}

func (s *AppConfigApiServiceServerImpl) toServerAppConfigItem(item *core.AppConfig) skeled.AppConfigItem {
	return toServerAppConfigItem(item, toServerFieldSources(item.FieldSources))
}

func toServerAppConfigItem(item *core.AppConfig, fieldSources []skeled.FieldSource) skeled.AppConfigItem {
	return skeled.AppConfigItem{
		Id:           item.Id,
		Key:          item.Name,
		Status:       string(item.Status),
		Lifecycle:    item.Lifecycle,
		Value:        item.Value,
		Schema:       toServerAppConfigSchema(item.Definition),
		FieldSources: fieldSources,
	}
}

func toServerAppConfigListItem(item *core.AppConfig) skeled.AppConfigListItem {
	schemaName := ""
	schemaSkelName := ""
	if item.Definition != nil {
		schemaName = item.Definition.Name
		schemaSkelName = item.Definition.SkelName
	}
	return skeled.AppConfigListItem{
		Id:             item.Id,
		Key:            item.Name,
		Status:         string(item.Status),
		Lifecycle:      item.Lifecycle,
		SchemaName:     schemaName,
		SchemaSkelName: schemaSkelName,
	}
}

// sortedAppConfigSlots keeps the dashboard ordering: mismatch, unconfigured,
// unused, then the remaining configs by key, with the newest unused first.
func sortedAppConfigSlots(slots []*core.AppConfig) []*core.AppConfig {
	return vslice.SortBy(slots, func(a *core.AppConfig, b *core.AppConfig) bool {
		aOrder := appConfigStatusOrder(a.Status)
		bOrder := appConfigStatusOrder(b.Status)
		if aOrder != bOrder {
			return aOrder < bOrder
		}
		if a.Status == core.AppConfigStatusUnused && !a.CreatedAt.Equal(b.CreatedAt) {
			return b.CreatedAt.Compare(a.CreatedAt) < 0
		}
		return cmp.Compare(a.Name, b.Name) < 0
	})
}

func appConfigStatusOrder(status core.AppConfigStatus) int {
	switch status {
	case core.AppConfigStatusMismatch:
		return 0
	case core.AppConfigStatusUnconfigured:
		return 1
	case core.AppConfigStatusUnused:
		return 2
	default:
		return 3
	}
}

func toServerAppConfigSchema(definition *core.AppConfigDefinition) *skeled.AppConfigSchema {
	if definition == nil {
		return nil
	}
	return &skeled.AppConfigSchema{
		SkelName:         definition.SkelName,
		Name:             definition.Name,
		Description:      definition.Description,
		Deprecated:       definition.Deprecated,
		DeprecatedReason: definition.DeprecatedReason,
		Lifecycle:        definition.Lifecycle,
		Fields:           toServerAppConfigSchemaFields(definition.Fields),
	}
}

func toServerAppConfigSchemaFields(fields []core.AppConfigField) []skeled.AppConfigSchemaField {
	ret := make([]skeled.AppConfigSchemaField, 0, len(fields))
	for _, field := range fields {
		ret = append(ret, skeled.AppConfigSchemaField{
			Name:              field.Name,
			Type:              field.Type,
			Description:       field.Description,
			Deprecated:        field.Deprecated,
			DeprecatedReason:  field.DeprecatedReason,
			EnumItems:         toServerAppConfigSchemaEnumItems(field.EnumItems),
			MapKeyEnumItems:   toServerAppConfigSchemaEnumItems(field.MapKeyEnumItems),
			MapValueEnumItems: toServerAppConfigSchemaEnumItems(field.MapValueEnumItems),
		})
	}
	return ret
}

func toServerAppConfigSchemaEnumItems(items []core.AppConfigEnumItem) []skeled.AppConfigSchemaEnumItem {
	ret := make([]skeled.AppConfigSchemaEnumItem, 0, len(items))
	for _, item := range items {
		ret = append(ret, skeled.AppConfigSchemaEnumItem{
			Name:             item.Name,
			Description:      item.Description,
			Deprecated:       item.Deprecated,
			DeprecatedReason: item.DeprecatedReason,
		})
	}
	return ret
}

// isValidConfigSkelName reports whether the Dashboard may create a value under
// the name: a dotted Skel name whose final segment is a Config name. Skel owns
// the identifier grammar, so this only guards the shape Hub serves.
func isValidConfigSkelName(skelName string) bool {
	segments := strings.Split(skelName, ".")
	if len(segments) < 2 {
		return false
	}

	if !strings.HasSuffix(segments[len(segments)-1], "Config") {
		return false
	}
	for _, segment := range segments {
		if !isSkelIdentifierSegment(segment) {
			return false
		}
	}
	return true
}

// isSkelIdentifierSegment matches one dot-separated part of a Skel name: ASCII
// letters, digits and underscores, never starting with a digit.
func isSkelIdentifierSegment(segment string) bool {
	for index, char := range segment {
		switch {
		case char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'):
		case index > 0 && char >= '0' && char <= '9':
		default:
			return false
		}
	}
	return segment != ""
}
