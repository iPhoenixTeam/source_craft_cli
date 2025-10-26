package cli

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

func printPrHelp() {
    fmt.Fprintln(os.Stderr, `pr commands:
  pr list <org> <repo> [--pageSize N] [--filter filter] [--pageToken QUERY]
  pr create <org> <repo> <title> <body> <branch>
  pr view <org> <repo> <id>
  pr merge <org> <repo> <id> <method> <squash> <deleteBranch> НЕ РЕАЛИЗОВАНО
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
			os.Exit(1)
			
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
	path := fmt.Sprintf("/repos/%s/%s/pulls", orgSlug, repoSlug)
	
	q := make([]string, 0, 3)
    if pageSize > 0 {
        q = append(q, fmt.Sprintf("page_size=%d", pageSize))
    }
    if pageToken != "" {
        q = append(q, "page_token="+url.QueryEscape(pageToken))
    }
    if filter != "" {
        q = append(q, "filter="+url.QueryEscape(filter))
    }
    if len(q) > 0 {
        path = path + "?" + strings.Join(q, "&")
    }

	resp, err := DoRequest("GET", path, nil)
	Ensure(err)

	items := JsonObject{}
	if result, ok := resp["pull_requests"].(JsonObject); ok {
		items = result
	}

	if len(items) == 0 {
		fmt.Printf("Pull requests %s/%s\n\n", orgSlug, repoSlug)
		fmt.Println("(no pull requests)")
		return
	}

	fmt.Printf("Pull requests %s/%s\n\n", orgSlug, repoSlug)
	for _, it := range items {
		if a, ok := it.(JsonObject); ok {
			PrintSinglePr(a)
		}
	}
}

func PrintSinglePr(pr JsonObject) {
	id := ToString(pr["id"]) + "/" + ToString(pr["slug"])
	title := ToString(pr["title"])

	author := ""
	if a, ok := pr["author"].(JsonObject); ok {
		author = ToString(a["id"]) + "/" + ToString(a["slug"])
	}

	state := ToString(pr["status"])
	updated := prettyTime(pr["updated_at"])
	source := ToString(pr["source_branch"])
	target := ToString(pr["target_branch"])

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

func CreatePr(orgSlug, repoSlug, title, sourceBranch, targetBranch string, silent bool) {
	if orgSlug == "" || repoSlug == "" {
		Ensure(fmt.Errorf("repo not specified"))
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls", orgSlug, repoSlug)
	payload := JsonObject{
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
		fmt.Printf("Pull request created: %s/%s\n", ToString(result["slug"]), ToString(result["id"]))
		if url, ok := result["clone_urls"]; ok {
			fmt.Printf("  http: %s, ssh\n", url)
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

	id := ToString(result["id"]) + "/" + ToString(result["slug"])
	title := ToString(result["title"])
	state := ToString(result["status"])
	author := ""
	if a, ok := result["author"].(JsonObject); ok {
		author = ToString(a["slug"]) + "/" + ToString(a["id"])
	}
	created := prettyTime(result["created_at"])
	updated := prettyTime(result["updated_at"])
	source := ToString(result["source_branch"])
	target := ToString(result["target_branch"])
	commits := ToString(result["commits_count"])
	comments := ToString(result["comments_count"])
	description := ToString(result["description"])

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
			if mm, ok := r.(JsonObject); ok {
				fmt.Printf("  - %s (%s)\n", ToString(mm["slug"]) + "/" + ToString(mm["id"]), ToString(mm["status"]))
			} else {
				fmt.Printf("  - %v\n", r)
			}
		}
		fmt.Println()
	}

	if linked, ok := result["linked_issues"].([]any); ok && len(linked) > 0 {
		fmt.Printf("Linked issues: %d\n\n", len(linked))
	}

	if checks, ok := result["checks"].(JsonObject); ok && len(checks) > 0 {
		fmt.Println("Checks:")
		for k, v := range checks {
			fmt.Printf("  %-20s %v\n", k, v)
		}
		fmt.Println()
	}
}

func MergePr(orgSlug, repoSlug, prSlug, method string, squash bool, deleteBranch bool) {
	fmt.Println("НЕ РЕАЛИЗОВАНО")
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