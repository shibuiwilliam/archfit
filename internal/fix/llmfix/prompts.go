package llmfix

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

const systemPrompt = `You are an assistant embedded in archfit, a tool that evaluates repository architecture for coding agents.

You are generating the content of a file that will be created to fix an architectural finding.
The file should be concise, useful, and specific to this project.
Do NOT include explanations or preamble — output ONLY the file content.
Do NOT use markdown code fences around the output.`

func buildEnrichPrompt(ruleID string, change fix.Change, repo model.RepoFacts) llm.Prompt {
	var user strings.Builder
	fmt.Fprintf(&user, "Generate the content for file %q.\n\n", change.Path)
	fmt.Fprintf(&user, "Rule: %s\n", ruleID)
	fmt.Fprintf(&user, "File action: %s\n", change.Action)
	fmt.Fprintf(&user, "Project root: %s\n", filepath.Base(repo.Root))

	if len(repo.Languages) > 0 {
		langs := make([]string, 0, len(repo.Languages))
		for lang, count := range repo.Languages {
			langs = append(langs, fmt.Sprintf("%s(%d)", lang, count))
		}
		sort.Strings(langs)
		fmt.Fprintf(&user, "Languages: %s\n", strings.Join(langs, ", "))
	}

	fmt.Fprintf(&user, "\nHere is the static template as a starting point:\n---\n%s\n---\n", string(change.Content))
	fmt.Fprintf(&user, "\nImprove this template to be specific to this project. Keep it under 80 lines.\n")

	body := user.String()
	if len(body) > llm.MaxUserBytes {
		body = body[:llm.MaxUserBytes] + "\n...[truncated]"
	}

	return llm.Prompt{
		System:          systemPrompt,
		User:            body,
		MaxOutputTokens: 600,
	}
}
