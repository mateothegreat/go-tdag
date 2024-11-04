package tdag

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

type TDag struct {
	Nodes         []*TNode
	Edges         []*TEdge
	Ctx           *TestContext
	SetupFns      []func(ctx *TestContext)
	TearDownFns   []func(ctx *TestContext)
	BeforeEachFns []func(ctx *TestContext)
	AfterEachFns  []func(ctx *TestContext)
}

type TNode struct {
	ID string
	Fn TestFn
}

type TEdge struct {
	Left  *TNode
	Right *TNode
}

type TestContext struct {
	Store *TStore
	T     *testing.T
}

type TestFn func(ctx *TestContext)

func NewTDag(t *testing.T) *TDag {
	return &TDag{
		Ctx: &TestContext{
			Store: NewStore(),
			T:     t,
		},
		Nodes:         []*TNode{},
		Edges:         []*TEdge{},
		SetupFns:      []func(ctx *TestContext){},
		TearDownFns:   []func(ctx *TestContext){},
		BeforeEachFns: []func(ctx *TestContext){},
		AfterEachFns:  []func(ctx *TestContext){},
	}
}

func (d *TDag) AddNode(id string, fn TestFn) *TNode {
	node := &TNode{
		ID: id,
		Fn: fn,
	}
	d.Nodes = append(d.Nodes, node)
	return node
}

func (d *TDag) AddEdge(from string, to ...string) ([]*TEdge, error) {
	var edges []*TEdge
	fromNode := d.findNodeByID(from)
	if fromNode == nil {
		return nil, fmt.Errorf("node %s does not exist", from)
	}

	for _, targetID := range to {
		targetNode := d.findNodeByID(targetID)
		if targetNode == nil {
			return nil, fmt.Errorf("node %s does not exist", targetID)
		}

		if d.createsCycle(fromNode, targetNode) {
			return nil, fmt.Errorf("adding edge from %s to %s would create a cycle", from, targetID)
		}

		edge := &TEdge{
			Left:  fromNode,
			Right: targetNode,
		}
		d.Edges = append(d.Edges, edge)
		edges = append(edges, edge)
	}
	return edges, nil
}

func (d *TDag) Setup(fn func(ctx *TestContext)) {
	d.SetupFns = append(d.SetupFns, fn)
}

func (d *TDag) TearDown(fn func(ctx *TestContext)) {
	d.TearDownFns = append(d.TearDownFns, fn)
}

func (d *TDag) BeforeEach(fn func(ctx *TestContext)) {
	d.BeforeEachFns = append(d.BeforeEachFns, fn)
}

func (d *TDag) AfterEach(fn func(ctx *TestContext)) {
	d.AfterEachFns = append(d.AfterEachFns, fn)
}

func (d *TDag) findNodeByID(id string) *TNode {
	for _, node := range d.Nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}

// RunTests runs the tests in topological order.
//
// Arguments:
//   - t: The testing object.
func (d *TDag) RunTests(t *testing.T) {
	// Create dependency graph and track in-degree for each node.
	inDegree := make(map[string]int)
	outEdges := make(map[string][]*TNode)

	// Initialize in-degree counts and build adjacency list.
	for _, node := range d.Nodes {
		inDegree[node.ID] = 0
	}

	// Build adjacency list.
	for _, edge := range d.Edges {
		inDegree[edge.Right.ID]++
		outEdges[edge.Left.ID] = append(outEdges[edge.Left.ID], edge.Right)
	}

	// Keep track of completed nodes.
	completed := make(map[string]bool)
	var completedMux sync.Mutex

	// Run tests in topological order.
	for {
		// Find nodes with no dependencies.
		var available []*TNode
		for _, node := range d.Nodes {
			completedMux.Lock()
			if !completed[node.ID] && inDegree[node.ID] == 0 {
				available = append(available, node)
			}
			completedMux.Unlock()
		}

		// If no nodes are available but we haven't processed all nodes, we have a cycle.
		if len(available) == 0 {
			var remaining []string
			completedMux.Lock()
			for _, node := range d.Nodes {
				if !completed[node.ID] {
					remaining = append(remaining, node.ID)
				}
			}
			completedMux.Unlock()

			if len(remaining) > 0 {
				t.Fatalf("Dependency cycle detected. Remaining nodes: %v", remaining)
			}
			break
		}

		// Run available tests
		var wg sync.WaitGroup
		for _, node := range available {
			wg.Add(1)
			go func(n *TNode) {
				defer wg.Done()
				t.Run(n.ID, func(t *testing.T) {
					// Run the test passed in.
					for _, fn := range d.BeforeEachFns {
						fn(d.Ctx)
					}
					n.Fn(d.Ctx)
					for _, fn := range d.AfterEachFns {
						fn(d.Ctx)
					}
					// Mark as completed and update dependencies.
					completedMux.Lock()
					completed[n.ID] = true
					for _, dependent := range outEdges[n.ID] {
						inDegree[dependent.ID]--
					}
					completedMux.Unlock()
				})
			}(node)
		}
		wg.Wait()
	}

	// Run teardown functions.
	for _, fn := range d.TearDownFns {
		fn(d.Ctx)
	}
}

// RunTo runs tests up to the given node.
//
// Arguments:
//   - id: The ID of the node to run tests up to.
//   - t: The testing object.
func (d *TDag) RunTo(id string, t *testing.T) {
	// First verify the target node exists
	targetNode := d.findNodeByID(id)
	if targetNode == nil {
		t.Fatalf("Node %s does not exist", id)
		return
	}

	// Build a set of nodes we need to run (target and its dependencies).
	requiredNodes := make(map[string]bool)
	d.collectDependencies(id, requiredNodes)

	// Create dependency graph and track in-degree for required nodes
	inDegree := make(map[string]int)
	outEdges := make(map[string][]*TNode)

	// Initialize in-degree counts only for required nodes.
	for _, node := range d.Nodes {
		if requiredNodes[node.ID] {
			inDegree[node.ID] = 0
		}
	}

	// Build adjacency list only for required nodes.
	for _, edge := range d.Edges {
		if requiredNodes[edge.Left.ID] && requiredNodes[edge.Right.ID] {
			inDegree[edge.Right.ID]++
			outEdges[edge.Left.ID] = append(outEdges[edge.Left.ID], edge.Right)
		}
	}

	// Keep track of completed nodes
	completed := make(map[string]bool)
	var completedMux sync.Mutex

	// Run setup functions.
	for _, fn := range d.SetupFns {
		fn(d.Ctx)
	}

	// Run tests in topological order.
	for {
		// Find nodes with no dependencies.
		var available []*TNode
		for _, node := range d.Nodes {
			// Only consider nodes that are required.
			if !requiredNodes[node.ID] {
				continue
			}
			completedMux.Lock()
			if !completed[node.ID] && inDegree[node.ID] == 0 {
				available = append(available, node)
			}
			completedMux.Unlock()
		}

		// If no nodes are available but we haven't processed all required nodes, we have a cycle.
		if len(available) == 0 {
			var remaining []string
			completedMux.Lock()
			for nodeID := range requiredNodes {
				if !completed[nodeID] {
					remaining = append(remaining, nodeID)
				}
			}
			completedMux.Unlock()

			if len(remaining) > 0 {
				t.Fatalf("Dependency cycle detected. Remaining nodes: %v", remaining)
			}
			break
		}

		// Run available tests.
		var wg sync.WaitGroup
		for _, node := range available {
			wg.Add(1)
			go func(n *TNode) {
				defer wg.Done()
				t.Run(n.ID, func(t *testing.T) {
					for _, fn := range d.BeforeEachFns {
						fn(d.Ctx)
					}
					n.Fn(d.Ctx)
					for _, fn := range d.AfterEachFns {
						fn(d.Ctx)
					}
					completedMux.Lock()
					completed[n.ID] = true
					for _, dependent := range outEdges[n.ID] {
						inDegree[dependent.ID]--
					}
					completedMux.Unlock()
				})
			}(node)
		}
		wg.Wait()
	}

	// Run teardown functions.
	for _, fn := range d.TearDownFns {
		fn(d.Ctx)
	}
}

// collectDependencies recursively collects all dependencies for a given node.
func (d *TDag) collectDependencies(nodeID string, collected map[string]bool) {
	// Mark this node as required.
	collected[nodeID] = true

	// Find all edges where this node is on the right (dependencies).
	for _, edge := range d.Edges {
		if edge.Right.ID == nodeID && !collected[edge.Left.ID] {
			d.collectDependencies(edge.Left.ID, collected)
		}
	}
}

func (d *TDag) createsCycle(from, to *TNode) bool {
	visited := make(map[string]bool)
	return d.detectCycle(from, to, visited)
}

func (d *TDag) detectCycle(start, target *TNode, visited map[string]bool) bool {
	if start == nil || target == nil {
		return false
	}
	if start.ID == target.ID {
		return true
	}
	visited[start.ID] = true
	for _, edge := range d.Edges {
		if edge.Left.ID == start.ID && !visited[edge.Right.ID] {
			if d.detectCycle(edge.Right, target, visited) {
				return true
			}
		}
	}
	visited[start.ID] = false
	return false
}

func (d *TDag) ToD2(path string) error {
	var builder strings.Builder
	visited := make(map[string]bool)
	edgesPrinted := make(map[string]bool)

	for _, node := range d.Nodes {
		if _, inGroup := visited[node.ID]; !inGroup {
			d.buildD2FromNode(node, visited, edgesPrinted, &builder)
		}
	}

	// Write D2 output to file
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(builder.String()); err != nil {
		return err
	}
	return nil
}

func (d *TDag) buildD2FromNode(node *TNode, visited, edgesPrinted map[string]bool, builder *strings.Builder) {
	if visited[node.ID] {
		return
	}
	visited[node.ID] = true
	for _, edge := range d.Edges {
		if edge.Left.ID == node.ID {
			edgeKey := fmt.Sprintf("%s->%s", edge.Left.ID, edge.Right.ID)
			if !edgesPrinted[edgeKey] {
				builder.WriteString(fmt.Sprintf("%s -> %s\n", edge.Left.ID, edge.Right.ID))
				edgesPrinted[edgeKey] = true
			}
			d.buildD2FromNode(edge.Right, visited, edgesPrinted, builder)
		}
	}
	visited[node.ID] = false
}
