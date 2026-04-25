package depgraph

import "testing"

func TestTransitiveReach_Chain(t *testing.T) {
	// A → B → C
	g := Graph{
		Nodes: []Node{
			{Package: "a", Files: 1},
			{Package: "b", Files: 1},
			{Package: "c", Files: 1},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	if got := g.TransitiveReach("a"); got != 2 {
		t.Errorf("TransitiveReach(a) = %d, want 2", got)
	}
	if got := g.TransitiveReach("b"); got != 1 {
		t.Errorf("TransitiveReach(b) = %d, want 1", got)
	}
	if got := g.TransitiveReach("c"); got != 0 {
		t.Errorf("TransitiveReach(c) = %d, want 0", got)
	}
}

func TestTransitiveReach_Diamond(t *testing.T) {
	// A → B, A → C, B → D, C → D
	g := Graph{
		Nodes: []Node{
			{Package: "a", Files: 1},
			{Package: "b", Files: 1},
			{Package: "c", Files: 1},
			{Package: "d", Files: 1},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}

	if got := g.TransitiveReach("a"); got != 3 {
		t.Errorf("TransitiveReach(a) = %d, want 3", got)
	}
}

func TestTransitiveReach_UnknownPackage(t *testing.T) {
	g := Graph{
		Nodes: []Node{{Package: "a", Files: 1}},
	}
	if got := g.TransitiveReach("unknown"); got != 0 {
		t.Errorf("TransitiveReach(unknown) = %d, want 0", got)
	}
}

func TestMaxReach(t *testing.T) {
	// A → B → C, D (isolated)
	g := Graph{
		Nodes: []Node{
			{Package: "a", Files: 1},
			{Package: "b", Files: 1},
			{Package: "c", Files: 1},
			{Package: "d", Files: 1},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	pkg, reach := g.MaxReach()
	if pkg != "a" || reach != 2 {
		t.Errorf("MaxReach() = (%q, %d), want (\"a\", 2)", pkg, reach)
	}
}

func TestMaxReach_Empty(t *testing.T) {
	g := Graph{}
	pkg, reach := g.MaxReach()
	if pkg != "" || reach != 0 {
		t.Errorf("MaxReach() = (%q, %d), want (\"\", 0)", pkg, reach)
	}
}

func TestPackageCount(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{Package: "a", Files: 1},
			{Package: "b", Files: 2},
		},
	}
	if got := g.PackageCount(); got != 2 {
		t.Errorf("PackageCount() = %d, want 2", got)
	}
}

func TestPackageCount_Empty(t *testing.T) {
	g := Graph{}
	if got := g.PackageCount(); got != 0 {
		t.Errorf("PackageCount() = %d, want 0", got)
	}
}
