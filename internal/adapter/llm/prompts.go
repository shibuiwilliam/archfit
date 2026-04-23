package llm

import (
	"fmt"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// BuildFindingPrompt constructs the LLM prompt for "explain this finding and
// propose a concrete fix". The system message sets archfit's expectations;
// the user message contains the rule + evidence.
//
// The prompt is intentionally terse. Gemini's flash models handle long
// system prompts fine, but short system prompts cache better in the in-run
// cache (both system and user are part of the cache key).
func BuildFindingPrompt(rule model.Rule, finding model.Finding, projectType []string) Prompt {
	var user strings.Builder
	fmt.Fprintf(&user, "Rule: %s — %s\n", rule.ID, rule.Title)
	fmt.Fprintf(&user, "Principle: %s   Severity: %s   Evidence: %s\n",
		rule.Principle, rule.Severity, rule.EvidenceStrength)
	fmt.Fprintf(&user, "Rule rationale: %s\n\n", oneLine(rule.Rationale))
	fmt.Fprintf(&user, "Finding path: %q\n", finding.Path)
	fmt.Fprintf(&user, "Finding message: %s\n", finding.Message)
	if len(finding.Evidence) > 0 {
		user.WriteString("Evidence:\n")
		writeEvidence(&user, finding.Evidence)
	}
	if len(projectType) > 0 {
		fmt.Fprintf(&user, "\nProject type: %s\n", strings.Join(projectType, ", "))
	}

	body := user.String()
	if len(body) > MaxUserBytes {
		body = body[:MaxUserBytes] + "\n...[truncated]"
	}

	return Prompt{
		System: findingSystemPrompt,
		User:   body,
		// Short by design. Long responses are less useful and more expensive.
		MaxOutputTokens: 400,
	}
}

// BuildRulePrompt constructs the prompt for `explain --with-llm <rule-id>` —
// a rule-level explanation that is not tied to a specific finding.
func BuildRulePrompt(rule model.Rule, projectType []string) Prompt {
	var user strings.Builder
	fmt.Fprintf(&user, "Rule: %s — %s\n", rule.ID, rule.Title)
	fmt.Fprintf(&user, "Principle: %s   Severity: %s   Evidence: %s\n",
		rule.Principle, rule.Severity, rule.EvidenceStrength)
	fmt.Fprintf(&user, "Rule rationale: %s\n", oneLine(rule.Rationale))
	fmt.Fprintf(&user, "Static remediation: %s\n", oneLine(rule.Remediation.Summary))
	if len(projectType) > 0 {
		fmt.Fprintf(&user, "Project type: %s\n", strings.Join(projectType, ", "))
	}

	body := user.String()
	if len(body) > MaxUserBytes {
		body = body[:MaxUserBytes] + "\n...[truncated]"
	}
	return Prompt{
		System:          ruleSystemPrompt,
		User:            body,
		MaxOutputTokens: 400,
	}
}

const findingSystemPrompt = `You are an assistant embedded in archfit, a tool that evaluates whether a repository is well-shaped for coding agents.

A rule has fired. Produce a short, actionable explanation in 3 parts:
1. WHY IT MATTERS HERE — one sentence tailored to the evidence.
2. CONCRETE FIX — 2–5 bullet points with specific file paths / commands / snippets where possible.
3. WHEN TO SUPPRESS — one sentence on when an ignore entry would be the right answer instead of a fix.

Do NOT invent file paths or commands that the evidence does not support.
Do NOT exceed 200 words.
Do NOT restate the rule's title. Assume the reader already has it.`

const ruleSystemPrompt = `You are an assistant embedded in archfit, a tool that evaluates whether a repository is well-shaped for coding agents.

Given a rule, produce a concise explanation in 2 parts:
1. WHAT THIS RULE REWARDS — the ideal repo state.
2. HOW TO GET THERE — 2–5 bullets with concrete suggestions.

Do NOT exceed 180 words. Assume the reader has already read the static rule doc.`

// oneLine collapses whitespace in rule rationale/remediation text so the
// prompt stays compact and stable.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// writeEvidence marshals the evidence map in a stable order. Ordering matters
// because the prompt is part of the cache key.
func writeEvidence(w *strings.Builder, ev map[string]any) {
	keys := make([]string, 0, len(ev))
	for k := range ev {
		keys = append(keys, k)
	}
	// Stable ordering for cache hit consistency.
	stableSort(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "  - %s: %v\n", k, ev[k])
	}
}

// stableSort is a tiny insertion sort; evidence maps are small (<20 keys) and
// this avoids pulling sort into this file.
func stableSort(s []string) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && s[j-1] > s[j] {
			s[j-1], s[j] = s[j], s[j-1]
			j--
		}
	}
}
