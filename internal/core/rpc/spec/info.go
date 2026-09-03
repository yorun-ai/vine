package spec

import (
	"encoding/json/v2"
	"reflect"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vpre"
)

type ServiceInfo interface {
	Name() string
	SkelName() string
	Hash() string

	ServerType() reflect.Type
	ERServerType() reflect.Type
	DefaultServerType() reflect.Type
	DefaultERServerType() reflect.Type
	WrapperERServerCtor() any

	ClientType() reflect.Type
	ClientCtor() any
	ERClientType() reflect.Type
	ERClientCtor() any

	Methods() []MethodInfo
}

type _ServiceInfo struct {
	name     string
	skelName string
	hash     string

	serverRegistered    bool
	serverType          reflect.Type
	erServerType        reflect.Type
	defaultServerType   reflect.Type
	defaultERServerType reflect.Type
	wrapperERServerCtor any

	clientRegistered bool
	clientType       reflect.Type
	clientCtor       any
	erClientType     reflect.Type
	erClientCtor     any

	methodRegistered bool
	methods          []MethodInfo
}

func (si *_ServiceInfo) Name() string {
	return si.name
}

func (si *_ServiceInfo) SkelName() string {
	return si.skelName
}

func (si *_ServiceInfo) Hash() string {
	return si.hash
}

func (si *_ServiceInfo) ServerType() reflect.Type {
	return si.serverType
}

func (si *_ServiceInfo) ERServerType() reflect.Type {
	return si.erServerType
}

func (si *_ServiceInfo) DefaultServerType() reflect.Type {
	return si.defaultServerType
}

func (si *_ServiceInfo) DefaultERServerType() reflect.Type {
	return si.defaultERServerType
}

func (si *_ServiceInfo) WrapperERServerCtor() any {
	return si.wrapperERServerCtor
}

func (si *_ServiceInfo) ClientType() reflect.Type {
	return si.clientType
}

func (si *_ServiceInfo) ClientCtor() any {
	return si.clientCtor
}

func (si *_ServiceInfo) ERClientType() reflect.Type {
	return si.erClientType
}

func (si *_ServiceInfo) ERClientCtor() any {
	return si.erClientCtor
}

func (si *_ServiceInfo) Methods() []MethodInfo {
	return append([]MethodInfo(nil), si.methods...)
}

type MethodInfo interface {
	Name() string
	SkelName() string

	Service() ServiceInfo
	FullURLPath() string

	HasArguments() bool
	NewArguments() any
	ArgumentsType() reflect.Type
	ArgumentsSensitive() bool
	ArgumentsContainsBinaryType() bool
	PositionArguments(arguments any) []any
	ValidateArguments(any) error
	CloneArguments(any) any

	HasResult() bool
	NewResult() any
	ResultType() reflect.Type
	ResultSensitive() bool
	ResultContainsBinaryType() bool
	ValidateResult(any) error
	CloneResult(any) any
}

type _MethodInfo struct {
	name     string
	skelName string

	fromedService ServiceInfo
	fullURLPath   string

	argumentsType               reflect.Type
	argumentsSensitive          bool
	argumentsContainsBinaryType bool
	argumentFieldInfos          []_ArgumentFieldInfo
	validateArguments           func(any) error
	cloneArguments              func(any) any

	resultType               reflect.Type
	resultSensitive          bool
	resultContainsBinaryType bool
	validateResult           func(any) error
	cloneResult              func(any) any
}

func (mi *_MethodInfo) Name() string {
	return mi.name
}

func (mi *_MethodInfo) SkelName() string {
	return mi.skelName
}

func (mi *_MethodInfo) Service() ServiceInfo {
	return mi.fromedService
}

func (mi *_MethodInfo) FullURLPath() string {
	return mi.fullURLPath
}

func (mi *_MethodInfo) HasArguments() bool {
	return mi.argumentsType != nil
}

func (mi *_MethodInfo) NewArguments() any {
	return reflect.New(mi.argumentsType).Interface()
}

func (mi *_MethodInfo) ArgumentsType() reflect.Type {
	return mi.argumentsType
}

func (mi *_MethodInfo) ArgumentsSensitive() bool {
	return mi.argumentsSensitive
}

func (mi *_MethodInfo) ArgumentsContainsBinaryType() bool {
	return mi.argumentsContainsBinaryType
}

func (mi *_MethodInfo) PositionArguments(arguments any) []any {
	if !mi.HasArguments() {
		return nil
	}

	argsValue := reflect.ValueOf(arguments).Elem()
	positionalArguments := make([]any, len(mi.argumentFieldInfos))
	for _, argFieldInfo := range mi.argumentFieldInfos {
		positionalArguments[argFieldInfo.ArgIndex] = argsValue.Field(argFieldInfo.FieldIndex).Interface()
	}
	return positionalArguments
}

func (mi *_MethodInfo) ValidateArguments(arguments any) error {
	return mi.validateArguments(arguments)
}

func (mi *_MethodInfo) CloneArguments(arguments any) any {
	if mi.cloneArguments != nil {
		return mi.cloneArguments(arguments)
	}
	target := mi.NewArguments()
	cloneValueByMarshaling(arguments, target, mi.ArgumentsContainsBinaryType(), mi.Service().SkelName())
	return target
}

func (mi *_MethodInfo) HasResult() bool {
	return mi.resultType != nil
}

func (mi *_MethodInfo) NewResult() any {
	return reflect.New(mi.resultType).Interface()
}

func (mi *_MethodInfo) ResultType() reflect.Type {
	return mi.resultType
}

func (mi *_MethodInfo) ResultSensitive() bool {
	return mi.resultSensitive
}

func (mi *_MethodInfo) ResultContainsBinaryType() bool {
	return mi.resultContainsBinaryType
}

func (mi *_MethodInfo) ValidateResult(result any) error {
	return mi.validateResult(result)
}

func (mi *_MethodInfo) CloneResult(result any) any {
	if mi.cloneResult != nil {
		return mi.cloneResult(result)
	}
	target := mi.NewResult()
	cloneValueByMarshaling(result, target, mi.ResultContainsBinaryType(), mi.Service().SkelName())
	return reflect.ValueOf(target).Elem().Interface()
}

// cloneValueByMarshaling supports service specs generated before typed clone
// hooks. Its codec round trip is a compatibility fallback, not part of the
// in-process value-isolation contract.
func cloneValueByMarshaling(source any, target any, containsBinaryType bool, serviceSkelName string) {
	encoder := skel.EncoderForSkelName(serviceSkelName)
	if containsBinaryType {
		vpre.MustNil(cbor.Unmarshal(encoder.MustMarshalCbor(source), target))
		return
	}
	vpre.MustNil(json.Unmarshal(encoder.MustMarshalJson(source), target))
}

type EmptyArguments struct{}

const (
	argTagName = "arg"
)

type _ArgumentFieldInfo struct {
	Name       string
	FieldIndex int
	ArgIndex   int
}

func buildArgumentFieldInfos(argsType reflect.Type) []_ArgumentFieldInfo {
	vpre.Check(argsType.Kind() == reflect.Struct, "rpc arguments type must be a struct, got %s", argsType)

	var argFields []_ArgumentFieldInfo
	seenIndexes := map[int]string{}
	for index := 0; index < argsType.NumField(); index++ {
		field := argsType.Field(index)
		tag, ok := field.Tag.Lookup(argTagName)
		vpre.Must(ok)
		labels := strings.Split(tag, ",")
		vpre.Must(len(labels) > 0)

		argIndex, err := strconv.Atoi(labels[0])
		vpre.MustNil(err)
		vpre.Check(argIndex >= 0 && argIndex < argsType.NumField(), "arg index %d out of range on %s.%s", argIndex, argsType, field.Name)
		if existingFieldName, exists := seenIndexes[argIndex]; exists {
			vpre.Panicf("duplicate arg index %d on %s.%s and %s.%s", argIndex, argsType, existingFieldName, argsType, field.Name)
		}
		seenIndexes[argIndex] = field.Name

		argField := _ArgumentFieldInfo{
			Name:       field.Name,
			FieldIndex: index,
			ArgIndex:   argIndex,
		}
		argFields = append(argFields, argField)
	}

	for expectedIndex := 0; expectedIndex < len(argFields); expectedIndex++ {
		vpre.CheckNotEmpty(seenIndexes[expectedIndex], "missing arg index %d on %s", expectedIndex, argsType)
	}
	return argFields
}

func noopValidateResult(any) error {
	return nil
}

func noopValidateArguments(any) error {
	return nil
}
