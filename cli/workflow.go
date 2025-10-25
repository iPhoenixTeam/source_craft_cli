package cli

import (
	"fmt"
	"strings"
)

func ListWorkflows(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/ci_workflows", orgSlug, repoSlug)
	resp, err := Execute1("GET", path, nil)
	Ensure(err)

	items := extractArrayCandidates(resp, "workflows", "items", "data", "ci_workflows")
	if len(items) == 0 {
		fmt.Printf("Workflows for %s/%s\n\n", orgSlug, repoSlug)
		fmt.Println("(no workflows)")
		return
	}

	fmt.Printf("Workflows for %s/%s\n\n", orgSlug, repoSlug)
	for _, it := range items {
		w, ok := it.(map[string]any)
		if !ok {
			continue
		}
		id := fmtString(w["id"], w["name"])
		name := fmtString(w["name"], w["title"])
		enabled := fmtString(w["enabled"])
		triggers := joinStringsFrom(w["triggers"])
		updated := prettyTimeShortAny(w["updated_at"])
		lastRun := extractLastRunSummary(w)

		fmt.Printf("%-14s  %-30s  %s\n", ShortID(id), TruncateString(name, 30), enabledStatusSymbol(enabled))
		if triggers != "" {
			fmt.Printf("  triggers: %s\n", triggers)
		}
		if lastRun != "" {
			fmt.Printf("  last run: %s\n", lastRun)
		}
		if updated != "" {
			fmt.Printf("  updated: %s\n", updated)
		}
		fmt.Println()
	}
}

func WorkflowStatus(orgSlug, repoSlug, workflowID string, limit int) {
	if limit <= 0 {
		limit = 5
	}
	path := fmt.Sprintf("repos/%s/%s/ci_workflows/%s/runs", orgSlug, repoSlug, workflowID)
	q := map[string]any{"page_size": limit}
	resp, err := Execute1("GET", path, q)
	Ensure(err)

	items := extractArrayCandidates(resp, "runs", "items", "data")
	if len(items) == 0 {
		fmt.Printf("No runs for workflow %s\n", workflowID)
		return
	}

	fmt.Printf("Runs for workflow %s (%s/%s)\n\n", workflowID, orgSlug, repoSlug)
	for _, it := range items {
		r, ok := it.(map[string]any)
		if !ok {
			continue
		}
		runID := fmtString(r["id"], r["run_id"])
		status := fmtString(r["status"], r["state"])
		conclusion := fmtString(r["conclusion"], r["result"])
		actor := ""
		if a, ok := r["actor"].(map[string]any); ok {
			actor = fmtString(a["slug"], a["id"])
		}
		created := prettyTimeShortAny(r["created_at"])
		duration := fmtString(r["duration"], r["elapsed"])

		fmt.Printf("%s  %s  %s\n", ShortID(runID), statusSymbol(status), strings.ToUpper(conclusion))
		fmt.Printf("  by: %-20s  started: %s  elapsed: %s\n", TruncateString(actor, 20), created, duration)
		// job summary if present
		if jobs, ok := r["jobs"].([]any); ok && len(jobs) > 0 {
			fmt.Printf("  jobs: %d\n", len(jobs))
		}
		fmt.Println()
	}
}

// WorkflowLogs получает и печатает логи выполнения workflow (попытка нескольких форм)
func WorkflowLogs(orgSlug, repoSlug, runID string) {
	// try repo-scoped run logs endpoint
	candidates := []string{
		fmt.Sprintf("repos/%s/%s/ci_workflows/runs/%s/logs", orgSlug, repoSlug, runID),
		fmt.Sprintf("repos/%s/%s/ci_workflows/%s/logs", orgSlug, repoSlug, runID),
		fmt.Sprintf("ci/runs/%s/logs", runID),
	}
	var resp map[string]any
	var err error
	for _, p := range candidates {
		resp, err = Execute1("GET", p, nil)
		if err == nil {
			break
		}
	}
	Ensure(err)

	// logs may be returned as string, array of lines, or jobs with logs
	if s, ok := resp["logs"].(string); ok && s != "" {
		fmt.Println(s)
		return
	}
	if arr, ok := resp["lines"].([]any); ok && len(arr) > 0 {
		for _, l := range arr {
			fmt.Println(fmt.Sprint(l))
		}
		return
	}
	// fallback: try jobs -> steps -> logs
	if jobs, ok := resp["jobs"].([]any); ok && len(jobs) > 0 {
		for _, j := range jobs {
			if jm, ok := j.(map[string]any); ok {
				fmt.Printf("Job: %s\n", fmtString(jm["name"], jm["id"]))
				if steps, ok := jm["steps"].([]any); ok {
					for _, s := range steps {
						if sm, ok := s.(map[string]any); ok {
							fmt.Printf("  Step: %s\n", fmtString(sm["name"], sm["id"]))
							if l, ok := sm["log"].(string); ok && l != "" {
								for _, ln := range strings.Split(strings.TrimRight(l, "\n"), "\n") {
									fmt.Printf("    %s\n", ln)
								}
							}
						}
					}
				}
				fmt.Println()
			}
		}
		return
	}

	// last resort: pretty-print entire response as JSON
	fmt.Println(ToJson(resp))
}

// WorkflowRun запускает workflow и печатает результат инициирования
func WorkflowRun(orgSlug, repoSlug, workflowID string, inputs map[string]any) {
	path := fmt.Sprintf("repos/%s/%s/ci_workflows/%s/trigger", orgSlug, repoSlug, workflowID)
	body := map[string]any{}
	if inputs != nil && len(inputs) > 0 {
		body["inputs"] = inputs
	}
	result, err := Execute1("POST", path, body)
	Ensure(err)

	// результат может содержать run id или url
	runID := fmtString(result["run_id"], result["id"])
	url := fmtString(result["url"], result["html_url"], result["web_url"])
	fmt.Printf("Workflow %s triggered\n", workflowID)
	if runID != "" {
		fmt.Printf("  run id: %s\n", runID)
	}
	if url != "" {
		fmt.Printf("  url: %s\n", url)
	}
	// if response contains run summary, print brief status
	if status := fmtString(result["status"], result["state"]); status != "" {
		fmt.Printf("  status: %s\n", status)
	}
	fmt.Println()
}

/* --- small helpers reused across package --- */

func enabledStatusSymbol(s string) string {
	if strings.ToLower(s) == "true" || strings.ToLower(s) == "enabled" || strings.ToLower(s) == "active" {
		return "enabled"
	}
	return "disabled"
}

func statusSymbol(state string) string {
	switch strings.ToLower(state) {
	case "running", "in_progress", "inprogress":
		return "▶ running"
	case "queued", "pending":
		return "⏳ queued"
	case "success", "passed", "completed":
		return "✔ success"
	case "failed", "error":
		return "✖ failed"
	default:
		return strings.ToLower(state)
	}
}

func extractLastRunSummary(w map[string]any) string {
	// try common fields
	if lr, ok := w["last_run"].(map[string]any); ok {
		id := fmtString(lr["id"], lr["run_id"])
		status := fmtString(lr["status"], lr["state"])
		t := prettyTimeShortAny(lr["started_at"])
		return fmt.Sprintf("%s %s %s", ShortID(id), statusSymbol(status), t)
	}
	// alternate fields
	if lastRunId := fmtString(w["last_run_id"], w["last_run_slug"]); lastRunId != "" {
		return ShortID(lastRunId)
	}
	return ""
}