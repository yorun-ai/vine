package impl

import (
	"cmp"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type SkeletonServiceServerImpl struct {
	skeled.DefaultSkeletonServiceServer

	SchemaRepo core.SchemaRepo `inject:""`
}

func (s *SkeletonServiceServerImpl) ListDomains() []skeled.SkeletonDomain {
	views := s.SchemaRepo.ListDomainSchemaViews()
	serviceVersionsByKey := skeletonSchemaVersionsByKey(s.SchemaRepo.ListServiceSchemaVersions())
	webVersionsByKey := skeletonSchemaVersionsByKey(s.SchemaRepo.ListWebSchemaVersions())
	ret := make([]skeled.SkeletonDomain, 0, len(views))
	for _, view := range views {
		domain := toServerSkeletonDomain(view, serviceVersionsByKey, webVersionsByKey)
		if domain.Total == 0 {
			continue
		}
		ret = append(ret, domain)
	}
	return sortedSkeletonDomains(ret)
}

func (s *SkeletonServiceServerImpl) ListActors() []skeled.SkeletonActorItem {
	views := s.SchemaRepo.ListDomainSchemaViews()
	serviceVersionsByKey := skeletonSchemaVersionsByKey(s.SchemaRepo.ListServiceSchemaVersions())
	webVersionsByKey := skeletonSchemaVersionsByKey(s.SchemaRepo.ListWebSchemaVersions())
	ret := make([]skeled.SkeletonActorItem, 0)
	for _, version := range s.SchemaRepo.ListActorSchemaVersions() {
		actor := toServerSkeletonActorItem(toSkeletonVersionFields(version), version.Schema)
		fillServerSkeletonActorItemAccess(&actor, views, serviceVersionsByKey, webVersionsByKey)
		ret = append(ret, actor)
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListConfigs() []skeled.SkeletonConfigItem {
	ret := make([]skeled.SkeletonConfigItem, 0)
	for _, version := range s.SchemaRepo.ListConfigSchemaVersions() {
		ret = append(ret, toServerSkeletonConfigItem(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListServices() []skeled.SkeletonServiceItem {
	ret := make([]skeled.SkeletonServiceItem, 0)
	for _, version := range s.SchemaRepo.ListServiceSchemaVersions() {
		ret = append(ret, toServerSkeletonServiceItem(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListResources() []skeled.SkeletonResourceItem {
	ret := make([]skeled.SkeletonResourceItem, 0)
	for _, version := range s.SchemaRepo.ListResourceSchemaVersions() {
		ret = append(ret, toServerSkeletonResourceItem(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListWebs() []skeled.SkeletonWebItem {
	ret := make([]skeled.SkeletonWebItem, 0)
	for _, version := range s.SchemaRepo.ListWebSchemaVersions() {
		ret = append(ret, toServerSkeletonWebItem(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListTasks() []skeled.SkeletonTask {
	ret := make([]skeled.SkeletonTask, 0)
	for _, version := range s.SchemaRepo.ListTaskSchemaVersions() {
		ret = append(ret, toServerSkeletonTask(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListEvents() []skeled.SkeletonEventItem {
	ret := make([]skeled.SkeletonEventItem, 0)
	for _, version := range s.SchemaRepo.ListEventSchemaVersions() {
		ret = append(ret, toServerSkeletonEventItem(toSkeletonVersionFields(version), version.Schema))
	}
	return ret
}

func (s *SkeletonServiceServerImpl) ListData() []skeled.SkeletonData {
	ret := make([]skeled.SkeletonData, 0)
	for _, version := range s.SchemaRepo.ListDataSchemaVersions() {
		ret = append(ret, toServerSkeletonData(toSkeletonVersionFields(version), version.Schema))
	}
	for _, version := range s.SchemaRepo.ListEnumSchemaVersions() {
		ret = append(ret, toServerSkeletonEnumData(toSkeletonVersionFields(version), version.Schema))
	}
	return sortedSkeletonData(ret)
}

func sortedSkeletonActorItems(items []skeled.SkeletonActorItem) []skeled.SkeletonActorItem {
	return vslice.SortBy(items, func(a skeled.SkeletonActorItem, b skeled.SkeletonActorItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonConfigItems(items []skeled.SkeletonConfigItem) []skeled.SkeletonConfigItem {
	return vslice.SortBy(items, func(a skeled.SkeletonConfigItem, b skeled.SkeletonConfigItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonServiceItems(items []skeled.SkeletonServiceItem) []skeled.SkeletonServiceItem {
	return vslice.SortBy(items, func(a skeled.SkeletonServiceItem, b skeled.SkeletonServiceItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonResourceItems(items []skeled.SkeletonResourceItem) []skeled.SkeletonResourceItem {
	return vslice.SortBy(items, func(a skeled.SkeletonResourceItem, b skeled.SkeletonResourceItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonWebItems(items []skeled.SkeletonWebItem) []skeled.SkeletonWebItem {
	return vslice.SortBy(items, func(a skeled.SkeletonWebItem, b skeled.SkeletonWebItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonTasks(items []skeled.SkeletonTask) []skeled.SkeletonTask {
	return vslice.SortBy(items, func(a skeled.SkeletonTask, b skeled.SkeletonTask) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonEventItems(items []skeled.SkeletonEventItem) []skeled.SkeletonEventItem {
	return vslice.SortBy(items, func(a skeled.SkeletonEventItem, b skeled.SkeletonEventItem) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonData(items []skeled.SkeletonData) []skeled.SkeletonData {
	return vslice.SortBy(items, func(a skeled.SkeletonData, b skeled.SkeletonData) bool {
		return compareSkeletonItemVersion(a.SkelName, a.IsMain, a.SchemaHash, b.SkelName, b.IsMain, b.SchemaHash) < 0
	})
}

func sortedSkeletonDomains(items []skeled.SkeletonDomain) []skeled.SkeletonDomain {
	return vslice.SortBy(items, func(a skeled.SkeletonDomain, b skeled.SkeletonDomain) bool {
		if a.Domain != b.Domain {
			return cmp.Compare(a.Domain, b.Domain) < 0
		}
		if a.IsMain != b.IsMain {
			return a.IsMain
		}
		return cmp.Compare(b.SchemaHash, a.SchemaHash) < 0
	})
}

func compareSkeletonItemVersion(aName string, aIsMain bool, aHash string, bName string, bIsMain bool, bHash string) int {
	if order := cmp.Compare(aName, bName); order != 0 {
		return order
	}
	if aIsMain != bIsMain {
		if aIsMain {
			return -1
		}
		return 1
	}
	return cmp.Compare(bHash, aHash)
}

type _SkeletonVersionFields struct {
	Domain           string
	SchemaHash       string
	MainSchemaHash   string
	IsMultiVersion   bool
	IsMain           bool
	DomainSchemaHash string
}

func toSkeletonVersionFields[T any](version core.SchemaVersion[T]) _SkeletonVersionFields {
	return _SkeletonVersionFields{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.MultiVersion,
		IsMain:           version.Main,
		DomainSchemaHash: version.DomainSchemaHash,
	}
}

func toServerSkeletonDomain(
	view core.DomainSchemaView,
	serviceVersionsByKey map[string]core.SchemaVersion[*skel.ServiceSchema],
	webVersionsByKey map[string]core.SchemaVersion[*skel.WebSchema],
) skeled.SkeletonDomain {
	version := view.DomainVersion
	domain := skeled.SkeletonDomain{
		Domain:         version.Schema.Domain,
		SchemaHash:     version.Schema.Hash,
		MainSchemaHash: version.MainSchemaHash,
		IsMultiVersion: version.MultiVersion,
		IsMain:         version.Main,
		Actors:         make([]skeled.SkeletonActorItem, 0, len(view.Actors)),
		Configs:        make([]skeled.SkeletonConfigItem, 0, len(view.Configs)),
		Services:       make([]skeled.SkeletonServiceItem, 0, len(view.Services)),
		Resources:      make([]skeled.SkeletonResourceItem, 0, len(view.Resources)),
		Data:           make([]skeled.SkeletonData, 0, len(view.Data)+len(view.Enums)),
		Webs:           make([]skeled.SkeletonWebItem, 0, len(view.Webs)),
		Tasks:          make([]skeled.SkeletonTask, 0, len(view.Tasks)),
		Events:         make([]skeled.SkeletonEventItem, 0, len(view.Events)),
	}
	for _, item := range view.Actors {
		actor := toServerSkeletonActorItem(toSkeletonVersionFields(item), item.Schema)
		fillServerSkeletonActorItemAccess(&actor, []core.DomainSchemaView{view}, serviceVersionsByKey, webVersionsByKey)
		domain.Actors = append(domain.Actors, actor)
	}
	for _, item := range view.Configs {
		domain.Configs = append(domain.Configs, toServerSkeletonConfigItem(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Services {
		domain.Services = append(domain.Services, toServerSkeletonServiceItem(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Resources {
		domain.Resources = append(domain.Resources, toServerSkeletonResourceItem(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Data {
		domain.Data = append(domain.Data, toServerSkeletonData(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Enums {
		domain.Data = append(domain.Data, toServerSkeletonEnumData(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Webs {
		domain.Webs = append(domain.Webs, toServerSkeletonWebItem(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Tasks {
		domain.Tasks = append(domain.Tasks, toServerSkeletonTask(toSkeletonVersionFields(item), item.Schema))
	}
	for _, item := range view.Events {
		domain.Events = append(domain.Events, toServerSkeletonEventItem(toSkeletonVersionFields(item), item.Schema))
	}
	domain.Actors = sortedSkeletonActorItems(domain.Actors)
	domain.Configs = sortedSkeletonConfigItems(domain.Configs)
	domain.Services = sortedSkeletonServiceItems(domain.Services)
	domain.Resources = sortedSkeletonResourceItems(domain.Resources)
	domain.Data = sortedSkeletonData(domain.Data)
	domain.Webs = sortedSkeletonWebItems(domain.Webs)
	domain.Tasks = sortedSkeletonTasks(domain.Tasks)
	domain.Events = sortedSkeletonEventItems(domain.Events)
	domain.Total = len(domain.Actors) + len(domain.Configs) + len(domain.Services) + len(domain.Resources) + len(domain.Data) + len(domain.Webs) + len(domain.Tasks) + len(domain.Events)
	return domain
}

func toServerSkeletonActorItem(version _SkeletonVersionFields, schema *skel.ActorSchema) skeled.SkeletonActorItem {
	vias := make([]string, 0, len(schema.Vias))
	for _, via := range schema.Vias {
		vias = append(vias, string(via))
	}
	return skeled.SkeletonActorItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		ActorVias:        vias,
		AuthEnabled:      schema.AuthEnabled,
		Credential:       toServerSkeletonActorData(version, schema.AuthCredential),
		Info:             toServerSkeletonActorData(version, schema.AuthInfo),
		AuthService:      toServerSkeletonActorService(version, schema.AuthService),
		PermEnabled:      schema.PermEnabled,
		PermService:      toServerSkeletonActorService(version, schema.PermService),
		PermMethod:       toServerSkeletonActorMethod(schema.PermMethod),
		Services:         []skeled.SkeletonServiceItem{},
		Webs:             []skeled.SkeletonWebItem{},
	}
}

func toServerSkeletonActorData(actorVersion _SkeletonVersionFields, schema *skel.DataSchema) *skeled.SkeletonData {
	if schema == nil {
		return nil
	}
	item := toServerSkeletonData(toSkeletonDerivedVersionFields(actorVersion, schema.Hash), schema)
	return &item
}

func toServerSkeletonActorService(actorVersion _SkeletonVersionFields, schema *skel.ServiceSchema) *skeled.SkeletonServiceItem {
	if schema == nil {
		return nil
	}
	item := toServerSkeletonServiceItem(toSkeletonDerivedVersionFields(actorVersion, schema.Hash), schema)
	return &item
}

func toServerSkeletonActorMethod(schema *skel.MethodSchema) *skeled.SkeletonMethod {
	if schema == nil {
		return nil
	}
	item := toServerSkeletonMethod(schema)
	return &item
}

func toSkeletonDerivedVersionFields(parent _SkeletonVersionFields, schemaHash string) _SkeletonVersionFields {
	return _SkeletonVersionFields{
		Domain:           parent.Domain,
		SchemaHash:       schemaHash,
		MainSchemaHash:   schemaHash,
		IsMultiVersion:   false,
		IsMain:           true,
		DomainSchemaHash: parent.DomainSchemaHash,
	}
}

func fillServerSkeletonActorItemAccess(
	actor *skeled.SkeletonActorItem,
	views []core.DomainSchemaView,
	serviceVersionsByKey map[string]core.SchemaVersion[*skel.ServiceSchema],
	webVersionsByKey map[string]core.SchemaVersion[*skel.WebSchema],
) {
	serviceKeys := map[string]struct{}{}
	webKeys := map[string]struct{}{}

	for _, view := range views {
		if !domainSchemaCanReferenceActorVersion(view.DomainVersion.Schema, actor.SkelName, actor.SchemaHash) {
			continue
		}
		for _, schema := range view.DomainVersion.Schema.Services {
			if !skeletonActorRefsContain(schema.Audiences, actor.SkelName) {
				continue
			}
			key := skeletonSchemaVersionKey(schema.SkelName, schema.Hash)
			if _, ok := serviceKeys[key]; ok {
				continue
			}
			version, ok := serviceVersionsByKey[key]
			if !ok {
				continue
			}
			serviceKeys[key] = struct{}{}
			actor.Services = append(actor.Services, toServerSkeletonServiceItem(toSkeletonVersionFields(version), version.Schema))
		}
		for _, schema := range view.DomainVersion.Schema.Webs {
			if !skeletonActorRefsContain(schema.Audiences, actor.SkelName) {
				continue
			}
			key := skeletonSchemaVersionKey(schema.SkelName, schema.Hash)
			if _, ok := webKeys[key]; ok {
				continue
			}
			version, ok := webVersionsByKey[key]
			if !ok {
				continue
			}
			webKeys[key] = struct{}{}
			actor.Webs = append(actor.Webs, toServerSkeletonWebItem(toSkeletonVersionFields(version), version.Schema))
		}
	}

	actor.Services = sortedSkeletonServiceItems(actor.Services)
	actor.Webs = sortedSkeletonWebItems(actor.Webs)
}

func domainSchemaCanReferenceActorVersion(schema *skel.DomainSchema, skelName string, schemaHash string) bool {
	hasActor := false
	for _, actor := range schema.Actors {
		if actor.SkelName != skelName {
			continue
		}
		hasActor = true
		if actor.Hash != schemaHash {
			continue
		}
		return true
	}
	return !hasActor
}

func skeletonActorRefsContain(refs []*skel.ActorAudienceSchema, skelName string) bool {
	for _, ref := range refs {
		if ref.SkelName == skelName {
			return true
		}
	}
	return false
}

func skeletonSchemaVersionKey(skelName string, schemaHash string) string {
	return skelName + "\x00" + schemaHash
}

func skeletonSchemaVersionsByKey[T any](versions []core.SchemaVersion[T]) map[string]core.SchemaVersion[T] {
	ret := make(map[string]core.SchemaVersion[T], len(versions))
	for _, version := range versions {
		ret[skeletonSchemaVersionKey(version.SkelName, version.SchemaHash)] = version
	}
	return ret
}

func toServerSkeletonServiceItem(version _SkeletonVersionFields, schema *skel.ServiceSchema) skeled.SkeletonServiceItem {
	return skeled.SkeletonServiceItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Pub:              schema.Pub,
		AuthMode:         string(schema.AuthMode),
		Require:          toServerSkeletonPermExpr(schema.Require),
		Actors:           toServerSkeletonActorRefs(schema.Audiences),
		Methods:          toServerSkeletonMethods(schema.Methods),
	}
}

func toServerSkeletonResourceItem(version _SkeletonVersionFields, schema *skel.ResourceSchema) skeled.SkeletonResourceItem {
	return skeled.SkeletonResourceItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Checks:           toServerSkeletonResourceChecks(schema.Checks),
		Actions:          toServerSkeletonResourceActions(schema.Actions),
		CheckService:     toServerSkeletonResourceCheckService(version, schema.CheckService),
	}
}

func toServerSkeletonResourceCheckService(resourceVersion _SkeletonVersionFields, schema *skel.ServiceSchema) *skeled.SkeletonServiceItem {
	if schema == nil {
		return nil
	}
	item := toServerSkeletonServiceItem(toSkeletonDerivedVersionFields(resourceVersion, schema.Hash), schema)
	return &item
}

func toServerSkeletonConfigItem(version _SkeletonVersionFields, schema *skel.ConfigSchema) skeled.SkeletonConfigItem {
	return skeled.SkeletonConfigItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Pub:              schema.Pub,
		Lifecycle:        schema.Lifecycle,
		Fields:           toServerSkeletonFields(schema.Members),
	}
}

func toServerSkeletonWebItem(version _SkeletonVersionFields, schema *skel.WebSchema) skeled.SkeletonWebItem {
	return skeled.SkeletonWebItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Actors:           toServerSkeletonActorRefs(schema.Audiences),
	}
}

func toServerSkeletonTask(version _SkeletonVersionFields, schema *skel.TaskSchema) skeled.SkeletonTask {
	triggers := make([]skeled.SkeletonTrigger, 0, len(schema.Triggers))
	for _, trigger := range schema.Triggers {
		triggers = append(triggers, skeled.SkeletonTrigger{
			Name:             trigger.Name,
			SkelName:         trigger.SkelName,
			Description:      optionalString(trigger.Description),
			InputDescription: optionalString(trigger.InputDescription),
			Example:          optionalString(trigger.Example),
			Arguments:        toServerSkeletonFields(trigger.Arguments),
		})
	}
	return skeled.SkeletonTask{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Triggers:         triggers,
	}
}

func toServerSkeletonEventItem(version _SkeletonVersionFields, schema *skel.EventSchema) skeled.SkeletonEventItem {
	return skeled.SkeletonEventItem{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Pub:              schema.Pub,
		Fields:           toServerSkeletonFields(schema.Members),
	}
}

func toServerSkeletonData(version _SkeletonVersionFields, schema *skel.DataSchema) skeled.SkeletonData {
	return skeled.SkeletonData{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Enum:             false,
		TypeParameters:   append([]string{}, schema.TypeParameters...),
		Fields:           toServerSkeletonFields(schema.Members),
		EnumItems:        []skeled.SkeletonEnumItem{},
	}
}

func toServerSkeletonEnumData(version _SkeletonVersionFields, schema *skel.EnumSchema) skeled.SkeletonData {
	return skeled.SkeletonData{
		Domain:           version.Domain,
		SchemaHash:       version.SchemaHash,
		MainSchemaHash:   version.MainSchemaHash,
		IsMultiVersion:   version.IsMultiVersion,
		IsMain:           version.IsMain,
		DomainSchemaHash: version.DomainSchemaHash,
		Name:             schema.Name,
		SkelName:         schema.SkelName,
		Description:      optionalString(schema.Description),
		Enum:             true,
		TypeParameters:   []string{},
		Fields:           []skeled.SkeletonField{},
		EnumItems:        toServerSkeletonEnumItems(schema.Items),
	}
}

func toServerSkeletonEnumItems(schemas []*skel.EnumItemSchema) []skeled.SkeletonEnumItem {
	ret := make([]skeled.SkeletonEnumItem, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonEnumItem{
			Name:        schema.Name,
			Description: optionalString(schema.Description),
		})
	}
	return ret
}

func toServerSkeletonActorRefs(refs []*skel.ActorAudienceSchema) []skeled.SkeletonActorRef {
	ret := make([]skeled.SkeletonActorRef, 0, len(refs))
	for _, ref := range refs {
		ret = append(ret, skeled.SkeletonActorRef{
			Name:     ref.Name,
			SkelName: ref.SkelName,
			Via:      optionalString(string(ref.Via)),
		})
	}
	return ret
}

func toServerSkeletonMethods(schemas []*skel.MethodSchema) []skeled.SkeletonMethod {
	ret := make([]skeled.SkeletonMethod, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, toServerSkeletonMethod(schema))
	}
	return ret
}

func toServerSkeletonMethod(schema *skel.MethodSchema) skeled.SkeletonMethod {
	return skeled.SkeletonMethod{
		Name:              schema.Name,
		SkelName:          schema.SkelName,
		Description:       optionalString(schema.Description),
		InputDescription:  optionalString(schema.InputDescription),
		OutputDescription: optionalString(schema.OutputDescription),
		Example:           optionalString(schema.Example),
		AuthMode:          string(schema.AuthMode),
		Require:           toServerSkeletonPermExpr(schema.Require),
		OutputExample:     optionalString(schema.OutputExample),
		Arguments:         toServerSkeletonFields(schema.Arguments),
		ResultType:        formatSkeletonType(schema.ResultType),
	}
}

func toServerSkeletonPermExpr(schema *skel.PermRequire) *skeled.SkeletonPermExpr {
	if schema == nil {
		return nil
	}
	return toServerSkeletonPermExprNode(schema.Expr)
}

func toServerSkeletonPermExprNode(schema *skel.PermExpr) *skeled.SkeletonPermExpr {
	if schema == nil {
		return nil
	}
	children := make([]skeled.SkeletonPermExpr, 0, len(schema.Children))
	for _, child := range schema.Children {
		childExpr := toServerSkeletonPermExprNode(child)
		if childExpr == nil {
			continue
		}
		children = append(children, *childExpr)
	}
	return &skeled.SkeletonPermExpr{
		Mode:     string(schema.Mode),
		Code:     optionalString(schema.Code),
		Check:    toServerSkeletonPermCheck(schema.Check),
		Children: children,
	}
}

func toServerSkeletonPermCheck(schema *skel.PermCheckInvocation) *skeled.SkeletonPermCheck {
	if schema == nil {
		return nil
	}
	return &skeled.SkeletonPermCheck{
		ResourceSkelName: schema.ResourceSkelName,
		ActionName:       schema.ActionName,
		CheckName:        schema.CheckName,
		ServiceSkelName:  schema.ServiceSkelName,
		MethodSkelName:   schema.MethodSkelName,
		Arguments:        toServerSkeletonPermCheckArguments(schema.Arguments),
	}
}

func toServerSkeletonPermCheckArguments(schemas []*skel.PermCheckArgument) []skeled.SkeletonPermCheckArgument {
	ret := make([]skeled.SkeletonPermCheckArgument, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonPermCheckArgument{
			Name:     schema.Name,
			JsonPath: schema.JsonPath,
			Type:     formatSkeletonType(schema.Type),
		})
	}
	return ret
}

func toServerSkeletonResourceActions(schemas []*skel.ResourceActionSchema) []skeled.SkeletonResourceAction {
	ret := make([]skeled.SkeletonResourceAction, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonResourceAction{
			Name:           schema.Name,
			PermissionCode: schema.PermissionCode,
			Description:    optionalString(schema.Description),
			Checks:         toServerSkeletonResourceChecks(schema.Checks),
		})
	}
	return ret
}

func toServerSkeletonResourceChecks(schemas []*skel.ResourceCheckSchema) []skeled.SkeletonResourceCheck {
	ret := make([]skeled.SkeletonResourceCheck, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonResourceCheck{
			Name:           schema.Name,
			MethodName:     schema.Method.Name,
			MethodSkelName: schema.Method.SkelName,
			Arguments:      toServerSkeletonFields(schema.Arguments),
		})
	}
	return ret
}

func toServerSkeletonFields(schemas []*skel.MemberSchema) []skeled.SkeletonField {
	ret := make([]skeled.SkeletonField, 0, len(schemas))
	for _, schema := range schemas {
		ret = append(ret, skeled.SkeletonField{
			Name:        schema.Name,
			Type:        formatSkeletonType(schema.Type),
			Description: optionalString(schema.Description),
			Example:     optionalString(schema.Example),
		})
	}
	return ret
}

func formatSkeletonType(typeSchema *skel.TypeSchema) string {
	if typeSchema == nil {
		return ""
	}
	var ret string
	switch typeSchema.Kind {
	case skel.TypeKindScalar:
		ret = string(typeSchema.Scalar)
	case skel.TypeKindEnum, skel.TypeKindData, skel.TypeKindConfig, skel.TypeKindEvent:
		ret = formatSkeletonNamedType(typeSchema)
		if len(typeSchema.TypeArguments) > 0 {
			args := make([]string, 0, len(typeSchema.TypeArguments))
			for _, arg := range typeSchema.TypeArguments {
				args = append(args, formatSkeletonType(arg))
			}
			ret += "<" + strings.Join(args, ", ") + ">"
		}
	case skel.TypeKindTypeParameter:
		ret = typeSchema.Name
		if ret == "" {
			ret = shortSkelName(typeSchema.SkelName)
		}
	case skel.TypeKindList:
		ret = "list<" + formatSkeletonType(typeSchema.Element) + ">"
	case skel.TypeKindMap:
		ret = "map<" + formatSkeletonType(typeSchema.Key) + ", " + formatSkeletonType(typeSchema.Value) + ">"
	default:
		ret = string(typeSchema.Kind)
	}
	if typeSchema.Nullable {
		ret += "?"
	}
	return ret
}

func formatSkeletonNamedType(typeSchema *skel.TypeSchema) string {
	if typeSchema.SkelName != "" {
		return typeSchema.SkelName
	}
	if typeSchema.Name != "" {
		return typeSchema.Name
	}
	return shortSkelName(typeSchema.SkelName)
}

func shortSkelName(skelName string) string {
	for i := len(skelName) - 1; i >= 0; i-- {
		if skelName[i] == '.' {
			return skelName[i+1:]
		}
	}
	return skelName
}
