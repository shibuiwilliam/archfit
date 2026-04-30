package score_test

import (
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

func TestContextSpanP50(t *testing.T) {
	tests := []struct {
		name string
		git  model.GitFacts
		want float64
	}{
		{
			name: "no commits",
			git:  model.GitFacts{},
			want: 0,
		},
		{
			name: "single commit",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{FilesChanged: 5},
			}},
			want: 5,
		},
		{
			name: "odd number of commits, median is middle",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{FilesChanged: 1},
				{FilesChanged: 10},
				{FilesChanged: 3},
			}},
			want: 3,
		},
		{
			name: "even number of commits, picks upper median",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{FilesChanged: 1},
				{FilesChanged: 2},
				{FilesChanged: 3},
				{FilesChanged: 4},
			}},
			want: 3, // index 2 of sorted [1,2,3,4]
		},
		{
			name: "all zero files changed",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{FilesChanged: 0},
				{FilesChanged: 0},
			}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.ContextSpanP50(tt.git)
			if m.Name != "context_span_p50" {
				t.Errorf("Name = %q, want context_span_p50", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
			if m.Principle != "P1" {
				t.Errorf("Principle = %q, want P1", m.Principle)
			}
		})
	}
}

func TestContextSpanP50_HighMedian(t *testing.T) {
	// Simulate a repo where commits typically touch 12 files (above threshold 8).
	git := model.GitFacts{RecentCommits: []model.Commit{
		{FilesChanged: 15},
		{FilesChanged: 10},
		{FilesChanged: 12},
		{FilesChanged: 14},
		{FilesChanged: 11},
	}}
	m := score.ContextSpanP50(git)
	if m.Value != 12 {
		t.Errorf("expected median 12, got %v", m.Value)
	}
}

func TestContextSpanP50_LowMedian(t *testing.T) {
	// Simulate a well-scoped repo where commits touch 3 files (below threshold).
	git := model.GitFacts{RecentCommits: []model.Commit{
		{FilesChanged: 2},
		{FilesChanged: 3},
		{FilesChanged: 4},
		{FilesChanged: 1},
		{FilesChanged: 3},
	}}
	m := score.ContextSpanP50(git)
	if m.Value != 3 {
		t.Errorf("expected median 3, got %v", m.Value)
	}
}

func TestContextSpanP50_MixedWithMerges(t *testing.T) {
	// Merge commits have FilesChanged=0, should be excluded from median.
	git := model.GitFacts{RecentCommits: []model.Commit{
		{FilesChanged: 5},
		{FilesChanged: 0}, // merge
		{FilesChanged: 3},
		{FilesChanged: 0}, // merge
		{FilesChanged: 7},
	}}
	m := score.ContextSpanP50(git)
	// sorted non-zero: [3, 5, 7], median index 1 → 5
	if m.Value != 5 {
		t.Errorf("expected median 5 (excluding merges), got %v", m.Value)
	}
}

func TestVerificationLatency(t *testing.T) {
	tests := []struct {
		name string
		cmds model.CommandFacts
		want float64
	}{
		{
			name: "no results",
			cmds: model.CommandFacts{},
			want: 0,
		},
		{
			name: "single result",
			cmds: model.CommandFacts{Results: []model.CommandResult{
				{Command: "make", DurationMS: 1500},
			}},
			want: 1.5,
		},
		{
			name: "multiple results, reports max",
			cmds: model.CommandFacts{Results: []model.CommandResult{
				{Command: "make", DurationMS: 1500},
				{Command: "go", DurationMS: 3000},
				{Command: "npm", DurationMS: 500},
			}},
			want: 3.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.VerificationLatency(tt.cmds)
			if m.Name != "verification_latency_s" {
				t.Errorf("Name = %q, want verification_latency_s", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
			if m.Principle != "P4" {
				t.Errorf("Principle = %q, want P4", m.Principle)
			}
		})
	}
}

func TestInvariantCoverage(t *testing.T) {
	tests := []struct {
		name     string
		findings []model.Finding
		rules    []model.Rule
		want     float64
	}{
		{
			name:  "no rules",
			rules: nil,
			want:  1,
		},
		{
			name:     "no findings",
			findings: nil,
			rules:    []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)},
			want:     1,
		},
		{
			name: "warn findings do not reduce coverage",
			findings: []model.Finding{
				{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
			},
			rules: []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)},
			want:  1,
		},
		{
			name: "error finding reduces coverage",
			findings: []model.Finding{
				{RuleID: "P1.LOC.001", Severity: model.SeverityError},
			},
			rules: []model.Rule{
				mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1),
				mkRule("P2.SPC.001", model.P2SpecFirst, model.SeverityWarn, 1),
			},
			want: 0.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.InvariantCoverage(tt.findings, tt.rules)
			if m.Name != "invariant_coverage" {
				t.Errorf("Name = %q, want invariant_coverage", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
		})
	}
}

func TestParallelConflictRate(t *testing.T) {
	tests := []struct {
		name string
		git  model.GitFacts
		want float64
	}{
		{
			name: "no commits",
			git:  model.GitFacts{},
			want: 0,
		},
		{
			name: "no merges",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{Subject: "feat: add thing"},
				{Subject: "fix: bug"},
			}},
			want: 0,
		},
		{
			name: "one merge in four commits",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{Subject: "feat: add thing"},
				{Subject: "Merge branch 'main'"},
				{Subject: "fix: bug"},
				{Subject: "docs: update"},
			}},
			want: 0.25,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.ParallelConflictRate(tt.git)
			if m.Name != "parallel_conflict_rate" {
				t.Errorf("Name = %q, want parallel_conflict_rate", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
		})
	}
}

func TestRollbackSignal(t *testing.T) {
	tests := []struct {
		name string
		git  model.GitFacts
		want float64
	}{
		{
			name: "no commits",
			git:  model.GitFacts{},
			want: 0,
		},
		{
			name: "no reverts",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{Subject: "feat: add thing"},
			}},
			want: 0,
		},
		{
			name: "one revert in five commits",
			git: model.GitFacts{RecentCommits: []model.Commit{
				{Subject: "feat: add thing"},
				{Subject: "Revert \"feat: add thing\""},
				{Subject: "fix: bug"},
				{Subject: "docs: update"},
				{Subject: "chore: clean"},
			}},
			want: 0.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.RollbackSignal(tt.git)
			if m.Name != "rollback_signal" {
				t.Errorf("Name = %q, want rollback_signal", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
			if m.Principle != "P6" {
				t.Errorf("Principle = %q, want P6", m.Principle)
			}
		})
	}
}

func TestBlastRadius(t *testing.T) {
	tests := []struct {
		name string
		dep  model.DepGraphFacts
		want float64
	}{
		{
			name: "single package",
			dep:  model.DepGraphFacts{PackageCount: 1, MaxReach: 0},
			want: 0,
		},
		{
			name: "no packages",
			dep:  model.DepGraphFacts{PackageCount: 0},
			want: 0,
		},
		{
			name: "reaches half",
			dep:  model.DepGraphFacts{PackageCount: 5, MaxReach: 2},
			want: 0.5,
		},
		{
			name: "reaches all",
			dep:  model.DepGraphFacts{PackageCount: 4, MaxReach: 3},
			want: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := score.BlastRadius(tt.dep)
			if m.Name != "blast_radius_score" {
				t.Errorf("Name = %q, want blast_radius_score", m.Name)
			}
			if m.Value != tt.want {
				t.Errorf("Value = %v, want %v", m.Value, tt.want)
			}
			if m.Principle != "P5" {
				t.Errorf("Principle = %q, want P5", m.Principle)
			}
		})
	}
}
