package vmath

import "go.yorun.ai/vine/util/vslice"

// Graph is a directed graph whose nodes are values of N.
// Node and edge iteration preserves insertion order where applicable.
type Graph[N comparable] struct {
	nodes    []N
	edges    map[N][]N
	selfRefs map[N]struct{}
}

// NewGraph creates an empty directed graph.
func NewGraph[N comparable]() *Graph[N] {
	return &Graph[N]{
		nodes:    []N{},
		edges:    map[N][]N{},
		selfRefs: map[N]struct{}{},
	}
}

// AddNode adds node and reports whether it was newly added.
func (g *Graph[N]) AddNode(node N) bool {
	if _, ok := g.edges[node]; ok {
		return false
	}

	g.nodes = append(g.nodes, node)
	g.edges[node] = []N{}
	return true
}

// HasNode reports whether node is present in g.
func (g *Graph[N]) HasNode(node N) bool {
	_, ok := g.edges[node]
	return ok
}

// AddEdge adds directed edges from from to each value in tos.
// Missing nodes are added automatically and duplicate edges are ignored.
func (g *Graph[N]) AddEdge(from N, tos ...N) {
	if !g.HasNode(from) {
		g.AddNode(from)
	}

	for _, to := range tos {
		if !g.HasNode(to) {
			g.AddNode(to)
		}
		if vslice.Contains(g.edges[from], to) {
			continue
		}
		g.edges[from] = append(g.edges[from], to)
		if from == to {
			g.selfRefs[from] = struct{}{}
		}
	}
}

// RemoveEdge removes the directed edge from from to to and reports whether it existed.
func (g *Graph[N]) RemoveEdge(from, to N) bool {
	targets, ok := g.edges[from]
	if !ok {
		return false
	}

	index := vslice.Index(targets, to)
	if index == -1 {
		return false
	}

	g.edges[from] = vslice.Delete(targets, index, index+1)
	if from == to {
		delete(g.selfRefs, from)
	}
	return true
}

// FindCycles returns the strongly connected components that contain a cycle.
func (g *Graph[N]) FindCycles() [][]N {
	op := &_TarjanOp[N]{
		graph: g.edges,
		nodes: make([]_TarjanNode, 0, len(g.edges)),
		index: make(map[N]int, len(g.edges)),
	}
	for v := range op.graph {
		if _, ok := op.index[v]; !ok {
			op.strongConnect(v)
		}
	}

	cycles := make([][]N, 0, len(op.output))
	for _, raw := range op.output {
		if len(raw) == 1 {
			if _, ok := g.selfRefs[raw[0]]; !ok {
				continue
			}
		}
		cycles = append(cycles, raw)
	}
	return cycles
}

// FindCyclePath returns one closed path whose first and last nodes are equal.
func (g *Graph[N]) FindCyclePath() []N {
	const (
		unvisited = iota
		visiting
		visited
	)
	states := map[N]int{}
	stack := []N{}
	stackIndexes := map[N]int{}

	var visit func(N) []N
	visit = func(node N) []N {
		states[node] = visiting
		stackIndexes[node] = len(stack)
		stack = append(stack, node)

		for _, target := range g.edges[node] {
			switch states[target] {
			case unvisited:
				if cycle := visit(target); cycle != nil {
					return cycle
				}
			case visiting:
				cycle := vslice.Clone(stack[stackIndexes[target]:])
				return append(cycle, target)
			}
		}

		stack = stack[:len(stack)-1]
		delete(stackIndexes, node)
		states[node] = visited
		return nil
	}

	for _, node := range g.nodes {
		if states[node] != unvisited {
			continue
		}
		if cycle := visit(node); cycle != nil {
			return cycle
		}
	}
	return nil
}

// FindPath returns one path from the given node to the first matching node.
func (g *Graph[N]) FindPath(from N, match func(N) bool) []N {
	if !g.HasNode(from) {
		return nil
	}

	visited := map[N]bool{}
	var visit func(N, []N) []N
	visit = func(node N, path []N) []N {
		if visited[node] {
			return nil
		}
		visited[node] = true
		path = append(path, node)
		if match(node) {
			return path
		}

		for _, target := range g.edges[node] {
			if result := visit(target, path); result != nil {
				return result
			}
		}
		return nil
	}
	return visit(from, nil)
}

// TopologicalSort returns nodes in topological order.
// The boolean result is false when the graph contains a cycle.
func (g *Graph[N]) TopologicalSort() ([]N, bool) {
	indegrees := make(map[N]int, len(g.nodes))
	for _, node := range g.nodes {
		indegrees[node] = 0
	}
	for _, from := range g.nodes {
		for _, to := range g.edges[from] {
			indegrees[to]++
		}
	}

	sources := make([]N, 0, len(g.nodes))
	for _, node := range g.nodes {
		if indegrees[node] == 0 {
			sources = append(sources, node)
		}
	}

	sorted := make([]N, 0, len(g.nodes))
	for len(sources) > 0 {
		node := sources[0]
		sources = sources[1:]
		sorted = append(sorted, node)

		for _, to := range g.edges[node] {
			indegrees[to]--
			if indegrees[to] == 0 {
				sources = append(sources, to)
			}
		}
	}

	return sorted, len(sorted) == len(g.nodes)
}

type _TarjanOp[N comparable] struct {
	graph  map[N][]N
	nodes  []_TarjanNode
	stack  []N
	index  map[N]int
	output [][]N
}

type _TarjanNode struct {
	lowLink int
	stacked bool
}

func (op *_TarjanOp[N]) strongConnect(v N) *_TarjanNode {
	index := len(op.nodes)
	op.index[v] = index
	op.stack = append(op.stack, v)
	op.nodes = append(op.nodes, _TarjanNode{lowLink: index, stacked: true})
	node := &op.nodes[index]

	for _, w := range op.graph[v] {
		i, seen := op.index[w]
		if !seen {
			n := op.strongConnect(w)
			if n.lowLink < node.lowLink {
				node.lowLink = n.lowLink
			}
		} else if op.nodes[i].stacked && i < node.lowLink {
			node.lowLink = i
		}
	}

	if node.lowLink == index {
		var vertices []N
		i := len(op.stack) - 1
		for {
			w := op.stack[i]
			stackIndex := op.index[w]
			op.nodes[stackIndex].stacked = false
			vertices = append(vertices, w)
			if stackIndex == index {
				break
			}
			i--
		}
		op.stack = op.stack[:i]
		op.output = append(op.output, vertices)
	}

	return node
}
