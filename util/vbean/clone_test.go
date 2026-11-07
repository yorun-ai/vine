package vbean

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testScalar struct {
	Value int
}

type testChild struct {
	Name    string
	Tags    []string
	Metrics map[string]int
}

type testParent struct {
	Id       int
	Name     string
	Child    *testChild
	Children []testChild
	Index    map[string]*testChild
	Scalar   testScalar
}

type testRecursive struct {
	Value    int
	Next     *testRecursive
	Children []testRecursive
}

type testEnvelope struct {
	Label   string
	Payload any
}

type testOpaque struct {
	hidden []string
}

func (opaque testOpaque) Hidden() []string {
	return opaque.hidden
}

type testConcurrent struct {
	Values []string
	Index  map[string][]int
}

// testArguments mirrors a generated Rpc arguments struct: an unexported type
// with exported members, which the runtime passes as any.
type testArguments struct {
	Key   string
	Names []string
}

func testParentValue() testParent {
	return testParent{
		Id:   1,
		Name: "root",
		Child: &testChild{
			Name:    "child",
			Tags:    []string{"a", "b"},
			Metrics: map[string]int{"hits": 1},
		},
		Children: []testChild{{
			Name: "first",
			Tags: []string{"x"},
		}},
		Index: map[string]*testChild{
			"first": {
				Name: "first",
				Tags: []string{"x"},
			},
		},
		Scalar: testScalar{Value: 7},
	}
}

func TestDeepCloneKeepsScalarsAndValueStructs(t *testing.T) {
	assert.Equal(t, 7, DeepClone(7))
	assert.Equal(t, "text", DeepClone("text"))
	assert.True(t, DeepClone(true))
	assert.Equal(t, testScalar{Value: 3}, DeepClone(testScalar{Value: 3}))
	assert.Equal(t, [3]int{1, 2, 3}, DeepClone([3]int{1, 2, 3}))
}

func TestDeepClonePreservesNilValues(t *testing.T) {
	assert.Nil(t, DeepClone([]string(nil)))
	assert.Nil(t, DeepClone(map[string]int(nil)))
	assert.Nil(t, DeepClone[*testChild](nil))
	assert.Nil(t, DeepClone[any](nil))
	assert.Nil(t, DeepClone(testEnvelope{Label: "bean"}).Payload)
}

func TestDeepCloneKeepsEmptyCollections(t *testing.T) {
	assert.NotNil(t, DeepClone([]string{}))
	assert.NotNil(t, DeepClone(map[string]int{}))
}

func TestDeepCloneIsolatesSliceAndMapData(t *testing.T) {
	byteSource := []byte{1, 2, 3}
	byteClone := DeepClone(byteSource)
	byteClone[0] = 9
	assert.Equal(t, []byte{1, 2, 3}, byteSource)

	textSource := []string{"a", "b"}
	textClone := DeepClone(textSource)
	textClone[0] = "changed"
	assert.Equal(t, []string{"a", "b"}, textSource)

	mapSource := map[string][]string{"key": {"a"}}
	mapClone := DeepClone(mapSource)
	mapClone["key"][0] = "changed"
	assert.Equal(t, map[string][]string{"key": {"a"}}, mapSource)
}

func TestDeepCloneIsolatesNestedBeans(t *testing.T) {
	source := testParentValue()
	cloned := DeepClone(source)

	cloned.Child.Name = "changed"
	cloned.Child.Tags[0] = "changed"
	cloned.Child.Metrics["hits"] = 99
	cloned.Children[0].Tags[0] = "changed"
	cloned.Index["first"].Tags[0] = "changed"
	cloned.Children = append(cloned.Children, testChild{Name: "extra"})

	assert.True(t, source.Child != cloned.Child)
	assert.Equal(t, "child", source.Child.Name)
	assert.Equal(t, []string{"a", "b"}, source.Child.Tags)
	assert.Equal(t, map[string]int{"hits": 1}, source.Child.Metrics)
	assert.Equal(t, []string{"x"}, source.Children[0].Tags)
	assert.Equal(t, []string{"x"}, source.Index["first"].Tags)
	assert.Len(t, source.Children, 1)
}

func TestDeepCloneIsolatesArrays(t *testing.T) {
	source := [2]testChild{
		{Name: "a", Tags: []string{"x"}},
		{Name: "b"},
	}
	cloned := DeepClone(source)
	cloned[0].Tags[0] = "changed"

	assert.Equal(t, []string{"x"}, source[0].Tags)
	assert.Equal(t, "b", cloned[1].Name)
}

func TestDeepCloneHandlesRecursiveTypes(t *testing.T) {
	source := testRecursive{
		Value: 1,
		Next: &testRecursive{
			Value:    2,
			Children: []testRecursive{{Value: 3, Children: []testRecursive{{Value: 4}}}},
		},
		Children: []testRecursive{{Value: 5, Next: &testRecursive{Value: 6}}},
	}
	cloned := DeepClone(source)

	cloned.Next.Value = 20
	cloned.Next.Children[0].Value = 30
	cloned.Next.Children[0].Children[0].Value = 40
	cloned.Children[0].Next.Value = 60

	assert.Equal(t, 2, source.Next.Value)
	assert.Equal(t, 3, source.Next.Children[0].Value)
	assert.Equal(t, 4, source.Next.Children[0].Children[0].Value)
	assert.Equal(t, 6, source.Children[0].Next.Value)
}

func TestDeepCloneClonesInterfacePayloads(t *testing.T) {
	source := testEnvelope{Label: "bean", Payload: []string{"a"}}
	cloned := DeepClone(source)
	cloned.Payload.([]string)[0] = "changed"

	assert.Equal(t, []string{"a"}, source.Payload.([]string))
	assert.Equal(t, "bean", DeepClone(testEnvelope{Label: "bean", Payload: 42}).Label)
	assert.Equal(t, 42, DeepClone(testEnvelope{Label: "bean", Payload: 42}).Payload)
}

func TestDeepCloneClonesAnyEntryPoint(t *testing.T) {
	source := []string{"a", "b"}
	cloned := DeepClone[any](source).([]string)
	cloned[0] = "changed"

	assert.Equal(t, []string{"a", "b"}, source)
}

func TestDeepCloneClonesUnexportedArgumentsType(t *testing.T) {
	source := testArguments{Key: "bean", Names: []string{"a"}}
	cloned := DeepClone[any](source).(testArguments)
	cloned.Names[0] = "changed"
	cloned.Key = "changed"

	assert.Equal(t, "bean", source.Key)
	assert.Equal(t, []string{"a"}, source.Names)
}

func TestDeepCloneCopiesUnexportedFieldsByValue(t *testing.T) {
	source := testOpaque{hidden: []string{"a"}}
	cloned := DeepClone(source)

	// Unexported reference-backed fields follow Go assignment semantics and stay
	// shared with the source. Generated data beans expose their members, so the
	// Rpc boundary never depends on cloning them.
	source.hidden[0] = "changed"
	assert.Equal(t, []string{"changed"}, cloned.Hidden())
}

func TestDeepCloneBuildsPlansConcurrently(t *testing.T) {
	source := testConcurrent{
		Values: []string{"a"},
		Index:  map[string][]int{"key": {1}},
	}

	var waitGroup sync.WaitGroup
	for range 16 {
		waitGroup.Go(func() {
			for range 50 {
				cloned := DeepClone(source)
				cloned.Values[0] = "changed"
				cloned.Index["key"][0] = 9

				assert.Equal(t, []string{"a"}, source.Values)
				assert.Equal(t, []int{1}, source.Index["key"])
			}
		})
	}
	waitGroup.Wait()
}

type testCertItem struct {
	Id                   int
	Name                 string
	Issuer               string
	Domains              []string
	PublicKeyBase64      string
	PrivateKeyConfigured bool
	Enabled              bool
}

func testCertItems(size int) []testCertItem {
	items := make([]testCertItem, size)
	for index := range items {
		items[index] = testCertItem{
			Id:                   index,
			Name:                 "cert",
			Issuer:               "letsencrypt",
			Domains:              []string{"a.example.com", "b.example.com", "c.example.com", "d.example.com", "e.example.com"},
			PublicKeyBase64:      "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			PrivateKeyConfigured: true,
			Enabled:              true,
		}
	}
	return items
}

func BenchmarkDeepCloneCertItems(b *testing.B) {
	source := testCertItems(50)

	b.ReportAllocs()
	for b.Loop() {
		_ = DeepClone(source)
	}
}
