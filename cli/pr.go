package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func printPrHelp() {
    fmt.Fprintln(os.Stderr, `pr commands:
  pr list <org> <repo> [--pageSize N] [--filter filter] [--pageToken QUERY]
  pr create <org> <repo> <title> <body> <branch>
  pr view <org> <repo> <id>
  pr merge <org> <repo> <id> <method> <squash> <deleteBranch>
Use "pr <command> --help" for command-specific flags.`)
}

func DispatchPr(command string, args []string) {
	switch command{	
		case "list":
			fs := NewCmd("pr list", "Usage: %s list <org> <repo> [--pageSize N] [--filter filter] [--pageToken QUERY]\n", flag.ContinueOnError)

			pageSize := fs.Int("pageSize", 30, "max items to list")
			filter := fs.String("filter", "", "pull request state filter: open|closed|all")
			pageToken := fs.String("pageToken", "", "search token or query")
			
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 2, "Usage: pr list <org> <repo>")

				ListPr(rem[0], rem[1], *filter, int64(*pageSize), *pageToken)
			}

		case "create":
			fs := NewCmd("pr create", "Usage: %s create <org> <repo> <title> <body> <branch>\n", flag.ContinueOnError)
			
			silent := fs.Bool("silent", false, "silent")

			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 5, "Usage: pr create <org> <repo> <title> <body> <branch>")

				CreatePr(rem[0], rem[1], rem[2], rem[3], rem[4], *silent)
			}
			
		case "view":
			fs := NewCmd("pr view", "Usage: %s view <org> <repo> <id>\n", flag.ContinueOnError)
			
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 3, "Usage: pr view <org> <repo> <id>")

				ViewPr(rem[0], rem[1], rem[2])
			}

		case "merge":
			fs := NewCmd("pr merge", "Usage: %s merge <org> <repo> <id> <method> <squash> <deleteBranch>\n", flag.ContinueOnError)
			
			squash := fs.Bool("squash", false, "use squash merge")
			deleteBranch := fs.Bool("deleteBranch", false, "delete source branch after merge")
    	
			if err := fs.Parse(args); err != nil {
				rem := Require(fs, 4, "Usage: pr merge <org> <repo> <id> <method> [--squash] [--deleteBranch]")

				MergePr(rem[0], rem[1], rem[2], rem[3], *squash, *deleteBranch)
			}
			
		case "--help", "-h", "help", "":
			printPrHelp()

		default:
			printPrHelp()
    }
}

func ListPr(orgSlug, repoSlug, filter string, pageSize int64, pageToken string) {
	path := candidatePrListPath(orgSlug, repoSlug)
	q := map[string]any{}
	if filter != "" {
		q["filter"] = filter
	}
	if pageSize > 0 {
		q["page_size"] = pageSize
	}
	if pageToken != "" {
		q["page_token"] = pageToken
	}

	resp, err := DoRequest("GET", path, q)
	Ensure(err)

	items := extractArrayCandidates(resp, "pull_requests", "pulls", "items", "data", "results")
	if len(items) == 0 {
		fmt.Printf("Pull requests %s/%s\n\n", orgSlug, repoSlug)
		fmt.Println("(no pull requests)")
		return
	}

	fmt.Printf("Pull requests %s/%s\n\n", orgSlug, repoSlug)
	for _, it := range items {
		pr, ok := it.(map[string]any)
		if !ok {
			continue
		}
		id := string(pr["id"]) + "/" + pr["slug"]
		title := fmtString(pr["title"], pr["name"])
		author := ""
		if a, ok := pr["author"].(map[string]any); ok {
			author = fmtString(a["slug"], a["id"])
		}
		state := fmtString(pr["status"], pr["state"], pr["status_slug"])
		updated := prettyTimeShortAny(pr["updated_at"])
		source := fmtString(pr["source_branch"], pr["head"])
		target := fmtString(pr["target_branch"], pr["base"])

		extra := []string{}
		if source != "" || target != "" {
			extra = append(extra, fmt.Sprintf("%s→%s", source, target))
		}
		if author != "" {
			extra = append(extra, "by:"+author)
		}
		if updated != "" {
			extra = append(extra, "updated:"+updated)
		}
		extraStr := ""
		if len(extra) > 0 {
			extraStr = "  — " + strings.Join(extra, "  ")
		}

		fmt.Printf("%s %s  %-70s%s\n", ShortID(id), prStateSymbol(state), TruncateString(title, 70), extraStr)
	}
}

func CreatePr(orgSlug, repoSlug, title, sourceBranch, targetBranch string, silent bool) {
	if orgSlug == "" || repoSlug == "" {
		Ensure(fmt.Errorf("repo not specified"))
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls", orgSlug, repoSlug)
	payload := map[string]any{
		"title":         title,
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
	}

	if silent {
		_, err := DoRequest("POST", path+"?silent=true", payload)
		Ensure(err)
	} else {
		result, err := DoRequest("POST", path, payload)
		Ensure(err)
		fmt.Printf("Pull request created: %s\n", fmtString(result["slug"], result["id"]))
		if url := fmtString(result["url"], result["html_url"], result["web_url"]); url != "" {
			fmt.Printf("  url: %s\n", url)
		}
	}
}

func ViewPr(orgSlug, repoSlug, prSlug string) {
	path := ""
	if orgSlug != "" && repoSlug != "" {
		path = fmt.Sprintf("/repos/%s/%s/pulls/%s", orgSlug, repoSlug, prSlug)
	} else {
		path = fmt.Sprintf("/pulls/id:%s", prSlug)
	}
	result, err := DoRequest("GET", path, nil)
	Ensure(err)

	id := fmtString(result["id"], result["slug"])
	title := fmtString(result["title"], result["name"])
	state := fmtString(result["status"], result["state"])
	author := ""
	if a, ok := result["author"].(map[string]any); ok {
		author = fmtString(a["slug"], a["id"])
	}
	created := prettyTimeShortAny(result["created_at"])
	updated := prettyTimeShortAny(result["updated_at"])
	source := fmtString(result["source_branch"], result["head"])
	target := fmtString(result["target_branch"], result["base"])
	commits := fmtString(result["commits_count"], result["commits"])
	comments := fmtString(result["comments_count"], result["comments"])
	description := fmtString(result["description"], result["body"])

	fmt.Printf("%s\n", title)
	fmt.Printf("pr %s  %s\n\n", id, strings.ToUpper(state))
	meta := []string{}
	if author != "" {
		meta = append(meta, "author:"+author)
	}
	if source != "" && target != "" {
		meta = append(meta, fmt.Sprintf("branch:%s→%s", source, target))
	}
	if commits != "" {
		meta = append(meta, "commits:"+commits)
	}
	if comments != "" {
		meta = append(meta, "comments:"+comments)
	}
	if created != "" {
		meta = append(meta, "created:"+created)
	}
	if updated != "" {
		meta = append(meta, "updated:"+updated)
	}
	if len(meta) > 0 {
		fmt.Println(strings.Join(meta, "  "))
		fmt.Println()
	}
	if description != "" {
		fmt.Println(IndentMultilineString(description, 2))
		fmt.Println()
	}

	if rv, ok := result["reviewers"].([]any); ok && len(rv) > 0 {
		fmt.Println("Reviewers:")
		for _, r := range rv {
			if mm, ok := r.(map[string]any); ok {
				fmt.Printf("  - %s (%s)\n", fmtString(mm["slug"], mm["id"]), fmtString(mm["state"], mm["status"]))
			} else {
				fmt.Printf("  - %v\n", r)
			}
		}
		fmt.Println()
	}

	if linked, ok := result["linked_issues"].([]any); ok && len(linked) > 0 {
		fmt.Printf("Linked issues: %d\n\n", len(linked))
	}

	if checks, ok := result["checks"].(map[string]any); ok && len(checks) > 0 {
		fmt.Println("Checks:")
		for k, v := range checks {
			fmt.Printf("  %-20s %v\n", k, v)
		}
		fmt.Println()
	}
}

func MergePr(orgSlug, repoSlug, prSlug, method string, squash bool, deleteBranch bool) {
	if orgSlug == "" || repoSlug == "" {
		Ensure(fmt.Errorf("repo not specified"))
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", orgSlug, repoSlug, prSlug)
	body := map[string]any{}
	if method != "" {
		body["method"] = method
	}
	if squash {
		body["squash"] = true
	}
	if deleteBranch {
		body["delete_branch"] = true
	}
	result, err := DoRequest("POST", path, body)
	Ensure(err)

	merged := false
	if b, ok := result["merged"].(bool); ok {
		merged = b
	}
	if merged {
		fmt.Printf("Pull request %s merged\n", prSlug)
		if url := fmtString(result["url"], result["html_url"], result["web_url"]); url != "" {
			fmt.Printf("  url: %s\n", url)
		}
	} else {
		if status := fmtString(result["status"], result["state"]); status != "" {
			fmt.Printf("Merge result: %s\n", status)
		} else {
			fmt.Println("Merge finished (no explicit confirmation returned)")
		}
	}
	fmt.Println()
}

func prStateSymbol(s string) string {
	switch strings.ToLower(s) {
	case "open", "opened":
		return "○ open(ed)"
	case "merged":
		return "◆ merged"
	case "closed", "declined":
		return "● closed/declined"
	case "draft":
		return "… draft"
	default:
		return "·"
	}
}