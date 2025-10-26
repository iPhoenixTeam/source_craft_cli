package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func printWorkflowHelp() {
    fmt.Fprintln(os.Stderr, `workflow commands:
  workflow list <org> <repo> [--size N] [--token TOKEN]
  workflow status <org> <repo> <workflowID> [--limit N]
  workflow logs <org> <repo> <runID>
  workflow run <org> <repo> <workflowID> [--input key=val,...]`)
}

func DispatchWorkflow(command string, args []string) {
    switch command {

    case "list":
        fs := NewCmd("workflow list", "Usage: %s <org> <repo>\n", flag.ContinueOnError)

        if err := fs.Parse(args); err == nil {
            rem := Require(fs, 2, "Usage: workflow list <org> <repo>")

            ListWorkflows(rem[0], rem[1])
        } 

    case "status":
        fs := NewCmd("workflow status", "Usage: %s <org> <repo> <workflowID> [--limit N]\n", flag.ContinueOnError)
        limit := fs.Int("limit", 5, "limit runs")
        
		if err := fs.Parse(args); err == nil {
            rem := Require(fs, 3, "Usage: workflow status <org> <repo> <workflowID> [--limit N]")
   
            WorkflowStatus(rem[0], rem[1], rem[2], *limit)
        }

    case "logs":
        fs := NewCmd("workflow logs", "Usage: %s <org> <repo> <runID>\n", flag.ContinueOnError)
        
		if err := fs.Parse(args); err == nil {
            rem := Require(fs, 3, "Usage: workflow logs <org> <repo> <runID>")

            WorkflowLogs(rem[0], rem[1], rem[2])
        }

    case "run":
        fs := NewCmd("workflow run", "Usage: %s <org> <repo> <workflowID> [--input key=val,...]\n", flag.ContinueOnError)
        inputs := fs.String("input", "", "comma-separated key=val inputs")
        if err := fs.Parse(args); err == nil {
            rem := Require(fs, 3, "Usage: workflow run <org> <repo> <workflowID> [--input key=val,...]")

            var in JsonObject
            if *inputs != "" {
                in = JsonObject{}
                for _, pair := range strings.Split(*inputs, ",") {
                    kv := strings.SplitN(pair, "=", 2)
                    if len(kv) == 2 {
                        in[kv[0]] = kv[1]
                    }
                }
            }
            WorkflowRun(rem[0], rem[1], rem[2], in)
            return
        } else {
            return
        }

    case "--help", "-h", "help", "":
        printWorkflowHelp()

    default:
        printWorkflowHelp()
    }
}

func ListWorkflows(orgSlug, repoSlug string) {
	
}

func WorkflowStatus(orgSlug, repoSlug, workflowID string, limit int) {
	if limit <= 0 {
		limit = 5
	}
	path := fmt.Sprintf("/repos/%s/%s/ci_workflows/%s/runs", orgSlug, repoSlug, workflowID)
	q := JsonObject{"page_size": limit}
	resp, err := DoRequest("GET", path, q)
	Ensure(err)

	items := extractArrayCandidates(resp, "runs", "items", "data")
	if len(items) == 0 {
		fmt.Printf("No runs for workflow %s\n", workflowID)
		return
	}

	fmt.Printf("Runs for workflow %s (%s/%s)\n\n", workflowID, orgSlug, repoSlug)
	for _, it := range items {
		r, ok := it.(JsonObject)
		if !ok {
			continue
		}
		runID := ToString(r["id"])
		status := ToString(r["status"])
		conclusion := ToString(r["conclusion"])
		actor := ""
		if a, ok := r["actor"].(JsonObject); ok {
			actor = ToString(a["slug"]) + "/" + ToString(a["id"])
		}
		created := prettyTime(r["created_at"])
		duration := ToString(r["duration"])

		fmt.Printf("%s  %s  %s\n", ShortID(runID), statusSymbol(status), strings.ToUpper(conclusion))
		fmt.Printf("  by: %-20s  started: %s  elapsed: %s\n", TruncateString(actor, 20), created, duration)

		if jobs, ok := r["jobs"].([]any); ok && len(jobs) > 0 {
			fmt.Printf("  jobs: %d\n", len(jobs))
		}
		fmt.Println()
	}
}

func WorkflowLogs(orgSlug, repoSlug, runID string) {

	candidates := []string{
		fmt.Sprintf("/repos/%s/%s/ci_workflows/runs/%s/logs", orgSlug, repoSlug, runID),
		fmt.Sprintf("/repos/%s/%s/ci_workflows/%s/logs", orgSlug, repoSlug, runID),
		fmt.Sprintf("/ci/runs/%s/logs", runID),
	}
	var resp JsonObject
	var err error
	for _, p := range candidates {
		resp, err = DoRequest("GET", p, nil)
		if err == nil {
			break
		}
	}
	Ensure(err)

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

	if jobs, ok := resp["jobs"].([]any); ok && len(jobs) > 0 {
		for _, j := range jobs {
			if jm, ok := j.(JsonObject); ok {
				fmt.Printf("Job: %s\n", ToString(jm["name"]) + "/" + ToString(jm["id"]))
				if steps, ok := jm["steps"].([]any); ok {
					for _, s := range steps {
						if sm, ok := s.(JsonObject); ok {
							fmt.Printf("  Step: %s\n", ToString(sm["name"]) + "/" + ToString(sm["id"]))
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

	fmt.Println(ToJson(resp))
}

func WorkflowRun(orgSlug, repoSlug, workflowID string, inputs JsonObject) {
	path := fmt.Sprintf("/%s/%s/ci_workflows/%s/trigger", orgSlug, repoSlug, workflowID)
	body := JsonObject{}
	if inputs != nil && len(inputs) > 0 {
		body["inputs"] = inputs
	}
	result, err := DoRequest("POST", path, body)
	Ensure(err)

	runID := ToString(result["id"])
	
	fmt.Printf("Workflow %s triggered\n", workflowID)
	if runID != "" {
		fmt.Printf("  run id: %s\n", runID)
	}

	if status := ToString(result["status"]); status != "" {
		fmt.Printf("  status: %s\n", status)
	}
	fmt.Println()
}

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

func extractLastRunSummary(w JsonObject) string {
	if lr, ok := w["last_run"].(JsonObject); ok {
		id := ToString(lr["id"],)
		status := ToString(lr["status"])
		t := prettyTime(lr["started_at"])
		return fmt.Sprintf("%s %s %s", ShortID(id), statusSymbol(status), t)
	}

	if lastRunId := ToString(w["last_run_id"]); lastRunId != "" {
		return ShortID(lastRunId)
	}
	return ""
}