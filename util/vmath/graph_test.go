package vmath

import (
	"testing"

	"go.yorun.ai/vine/util/vslice"

	"github.com/stretchr/testify/assert"
)

func normalizeCycles[N cmpOrdered](cycles [][]N) [][]N {
	result := make([][]N, 0, len(cycles))
	for _, cycle := range cycles {
		result = append(result, vslice.Sort(cycle))
	}
	return vslice.SortBy(result, func(left []N, right []N) bool {
		minLen := len(left)
		if len(right) < minLen {
			minLen = len(right)
		}
		for i := 0; i < minLen; i++ {
			if left[i] != right[i] {
				return left[i] < right[i]
			}
		}
		return len(left) < len(right)
	})
}

type cmpOrdered interface {
	~int | ~string
}

func TestGraphFindCycles(t *testing.T) {
	graph := NewGraph[string]()
	graph.AddEdge("a", "b")
	graph.AddEdge("b", "c")
	graph.AddEdge("c", "a")
	graph.AddEdge("d", "d")
	graph.AddEdge("e", "f")

	cycles := normalizeCycles(graph.FindCycles())
	assert.Equal(t, [][]string{{"a", "b", "c"}, {"d"}}, cycles)
}

func TestGraphFindCyclePath(t *testing.T) {
	graph := NewGraph[string]()
	graph.AddEdge("root", "a")
	graph.AddEdge("a", "b")
	graph.AddEdge("b", "c")
	graph.AddEdge("c", "a")

	assert.Equal(t, []string{"a", "b", "c", "a"}, graph.FindCyclePath())
}

func TestGraphFindCyclePathSupportsSelfReference(t *testing.T) {
	graph := NewGraph[string]()
	graph.AddEdge("self", "self")

	assert.Equal(t, []string{"self", "self"}, graph.FindCyclePath())
}

func TestGraphFindPath(t *testing.T) {
	graph := NewGraph[string]()
	graph.AddEdge("root", "a", "other")
	graph.AddEdge("a", "b")
	graph.AddEdge("b", "target")

	assert.Equal(t,
		[]string{"root", "a", "b", "target"},
		graph.FindPath("root", func(node string) bool { return node == "target" }),
	)
	assert.Nil(t, graph.FindPath("other", func(node string) bool { return node == "target" }))
	assert.Nil(t, graph.FindPath("missing", func(string) bool { return true }))
}

func TestGraphIgnoresAcyclicSingleton(t *testing.T) {
	graph := NewGraph[int]()
	graph.AddEdge(1, 2)
	graph.AddEdge(2, 3)

	assert.Empty(t, graph.FindCycles())
}

func TestGraphTopologicalSort(t *testing.T) {
	graph := NewGraph[string]()

	assert.True(t, graph.AddNode("a"))
	assert.True(t, graph.AddNode("b"))
	assert.True(t, graph.AddNode("c"))
	assert.True(t, graph.AddNode("d"))
	assert.False(t, graph.AddNode("a"))

	graph.AddEdge("a", "b")
	graph.AddEdge("a", "c")
	graph.AddEdge("b", "d")
	graph.AddEdge("c", "d")

	sorted, ok := graph.TopologicalSort()
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b", "c", "d"}, sorted)
}

func TestGraphRemoveEdgeAndCycle(t *testing.T) {
	graph := NewGraph[string]()
	graph.AddNode("a")
	graph.AddNode("b")

	graph.AddEdge("a", "b")
	assert.True(t, graph.RemoveEdge("a", "b"))
	sorted, ok := graph.TopologicalSort()
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, sorted)
	assert.False(t, graph.RemoveEdge("missing", "b"))

	cyclic := NewGraph[string]()
	cyclic.AddNode("a")
	cyclic.AddNode("b")
	cyclic.AddEdge("a", "b")
	cyclic.AddEdge("b", "a")

	sorted, ok = cyclic.TopologicalSort()
	assert.False(t, ok)
	assert.Empty(t, sorted)
}

func FuzzGraphPaths(f *testing.F) {
	f.Add([]byte{4, 0, 1, 1, 2, 2, 0})
	f.Add([]byte{5, 0, 1, 0, 2, 2, 3, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		nodeCount := int(data[0]%12) + 1
		graph := NewGraph[int]()
		for node := 0; node < nodeCount; node++ {
			graph.AddNode(node)
		}
		limit := len(data)
		if limit > 257 {
			limit = 257
		}
		for index := 1; index+1 < limit; index += 2 {
			from := int(data[index]) % nodeCount
			to := int(data[index+1]) % nodeCount
			graph.AddEdge(from, to)
		}

		_, acyclic := graph.TopologicalSort()
		cycle := graph.FindCyclePath()
		if acyclic != (cycle == nil) {
			t.Fatalf("topological and cycle results disagree: acyclic=%t cycle=%v", acyclic, cycle)
		}
		if cycle != nil {
			if len(cycle) < 2 || cycle[0] != cycle[len(cycle)-1] {
				t.Fatalf("cycle is not closed: %v", cycle)
			}
			for index := 0; index < len(cycle)-1; index++ {
				if !vslice.Contains(graph.edges[cycle[index]], cycle[index+1]) {
					t.Fatalf("cycle contains missing edge: %v", cycle)
				}
			}
		}

		from := int(data[0]) % nodeCount
		target := int(data[len(data)-1]) % nodeCount
		path := graph.FindPath(from, func(node int) bool { return node == target })
		if graphReachable(graph, from, target) != (path != nil) {
			t.Fatalf("path reachability mismatch: from=%d target=%d path=%v", from, target, path)
		}
		if path != nil {
			if path[0] != from || path[len(path)-1] != target {
				t.Fatalf("path has invalid endpoints: %v", path)
			}
			for index := 0; index < len(path)-1; index++ {
				if !vslice.Contains(graph.edges[path[index]], path[index+1]) {
					t.Fatalf("path contains missing edge: %v", path)
				}
			}
		}
	})
}

func graphReachable(graph *Graph[int], from, target int) bool {
	queue := []int{from}
	visited := map[int]bool{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if visited[node] {
			continue
		}
		visited[node] = true
		if node == target {
			return true
		}
		queue = append(queue, graph.edges[node]...)
	}
	return false
}
