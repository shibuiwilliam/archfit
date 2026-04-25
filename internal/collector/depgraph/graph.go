// Package depgraph builds package-level dependency graphs from source files.
//
// The graph is language-agnostic at this layer. Language-specific parsers
// (e.g. CollectGo) populate it. Resolvers use the graph to compute metrics
// such as blast_radius_score and context_span_p50.
package depgraph

// Graph represents a package-level dependency graph.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node is a package in the graph.
type Node struct {
	Package string `json:"package"`
	Files   int    `json:"files"`
}

// Edge is a directed import from one package to another.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TransitiveReach returns the number of packages transitively reachable from pkg.
// It performs a BFS over outgoing edges. If pkg is not in the graph, it returns 0.
func (g *Graph) TransitiveReach(pkg string) int {
	adj := g.adjacency()
	visited := map[string]bool{}
	queue := []string{pkg}
	visited[pkg] = true

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	// Subtract 1 because we don't count the starting package itself.
	reach := len(visited) - 1
	if reach < 0 {
		return 0
	}
	return reach
}

// MaxReach returns the package with the highest transitive reach and its reach
// count. If the graph has no nodes, it returns ("", 0).
func (g *Graph) MaxReach() (pkg string, reach int) {
	best := ""
	bestReach := -1
	for _, n := range g.Nodes {
		r := g.TransitiveReach(n.Package)
		if r > bestReach {
			bestReach = r
			best = n.Package
		}
	}
	if bestReach < 0 {
		return "", 0
	}
	return best, bestReach
}

// PackageCount returns the number of nodes.
func (g *Graph) PackageCount() int {
	return len(g.Nodes)
}

// adjacency builds an adjacency list from edges.
func (g *Graph) adjacency() map[string][]string {
	adj := make(map[string][]string, len(g.Nodes))
	for _, e := range g.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	return adj
}
