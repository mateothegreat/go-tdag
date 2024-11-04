package dag

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

type Dag[T any] struct {
	Nodes   []*Node
	Edges   []*Edge
	Context T
	Store   *Store
}

type Node struct {
	ID string
	Fn func(t *testing.T)
}

type Edge struct {
	Left  *Node
	Right *Node
}

type Store struct {
	items map[string]interface{}
}

func (s *Store) Set(key string, value interface{}) {
	s.items[key] = value
}

func (s *Store) Get(key string) interface{} {
	return s.items[key]
}

func NewStore() *Store {
	return &Store{
		items: make(map[string]interface{}),
	}
}

func NewDag[T any]() *Dag[T] {
	return &Dag[T]{
		Context: *new(T),
		Store:   NewStore(),
		Nodes:   []*Node{},
		Edges:   []*Edge{},
	}
}

func (d *Dag[T]) AddNode(id string, fn func(t *testing.T)) *Node {
	node := &Node{ID: id, Fn: fn}
	d.Nodes = append(d.Nodes, node)
	return node
}

func (d *Dag[T]) AddEdge(from string, to ...string) ([]*Edge, error) {
	var edges []*Edge
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

		edge := &Edge{
			Left:  fromNode,
			Right: targetNode,
		}
		d.Edges = append(d.Edges, edge)
		edges = append(edges, edge)
	}
	return edges, nil
}

func (d *Dag[T]) findNodeByID(id string) *Node {
	for _, node := range d.Nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}

func (d *Dag[T]) Test(t *testing.T) {
	for _, node := range d.Nodes {
		node.Fn(t)
	}
}

func (d *Dag[T]) createsCycle(from, to *Node) bool {
	visited := make(map[string]bool)
	return d.detectCycle(from, to, visited)
}

func (d *Dag[T]) detectCycle(start, target *Node, visited map[string]bool) bool {
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

func (d *Dag[T]) toD2(path string) error {
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

func (d *Dag[T]) buildD2FromNode(node *Node, visited, edgesPrinted map[string]bool, builder *strings.Builder) {
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

// RunTo runs tests up to the given node.
//
// Arguments:
//   - id: The ID of the node to run tests up to.
//   - t: The testing object.
func (d *Dag[T]) RunTo(id string, t *testing.T) {
	// First verify the target node exists
	targetNode := d.findNodeByID(id)
	if targetNode == nil {
		t.Fatalf("Node %s does not exist", id)
		return
	}

	// Build a set of nodes we need to run (target and its dependencies)
	requiredNodes := make(map[string]bool)
	d.collectDependencies(id, requiredNodes)

	// Create dependency graph and track in-degree for required nodes
	inDegree := make(map[string]int)
	outEdges := make(map[string][]*Node)

	// Initialize in-degree counts only for required nodes
	for _, node := range d.Nodes {
		if requiredNodes[node.ID] {
			inDegree[node.ID] = 0
		}
	}

	// Build adjacency list only for required nodes
	for _, edge := range d.Edges {
		if requiredNodes[edge.Left.ID] && requiredNodes[edge.Right.ID] {
			inDegree[edge.Right.ID]++
			outEdges[edge.Left.ID] = append(outEdges[edge.Left.ID], edge.Right)
		}
	}

	// Keep track of completed nodes
	completed := make(map[string]bool)
	var completedMux sync.Mutex

	// Run tests in topological order
	for {
		// Find nodes with no dependencies
		var available []*Node
		for _, node := range d.Nodes {
			// Only consider nodes that are required
			if !requiredNodes[node.ID] {
				continue
			}
			completedMux.Lock()
			if !completed[node.ID] && inDegree[node.ID] == 0 {
				available = append(available, node)
			}
			completedMux.Unlock()
		}

		// If no nodes are available but we haven't processed all required nodes, we have a cycle
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

		// Run available tests
		var wg sync.WaitGroup
		for _, node := range available {
			wg.Add(1)
			go func(n *Node) {
				defer wg.Done()
				t.Run(n.ID, func(t *testing.T) {
					n.Fn(t)
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
}

// collectDependencies recursively collects all dependencies for a given node
func (d *Dag[T]) collectDependencies(nodeID string, collected map[string]bool) {
	// Mark this node as required
	collected[nodeID] = true

	// Find all edges where this node is on the right (dependencies)
	for _, edge := range d.Edges {
		if edge.Right.ID == nodeID && !collected[edge.Left.ID] {
			d.collectDependencies(edge.Left.ID, collected)
		}
	}
}

// RunTests runs the tests in topological order.
//
// Arguments:
//   - t: The testing object.
func (d *Dag[T]) RunTests(t *testing.T) {
	// Create dependency graph and track in-degree for each node
	inDegree := make(map[string]int)
	outEdges := make(map[string][]*Node)

	// Initialize in-degree counts and build adjacency list
	for _, node := range d.Nodes {
		inDegree[node.ID] = 0
	}

	// Build adjacency list.
	for _, edge := range d.Edges {
		inDegree[edge.Right.ID]++
		outEdges[edge.Left.ID] = append(outEdges[edge.Left.ID], edge.Right)
	}

	// Keep track of completed nodes
	completed := make(map[string]bool)
	var completedMux sync.Mutex

	// Run tests in topological order.
	for {
		// Find nodes with no dependencies.
		var available []*Node
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
			go func(n *Node) {
				defer wg.Done()
				t.Run(n.ID, func(t *testing.T) {
					// Run the test passed in.
					n.Fn(t)
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
}
