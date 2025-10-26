package cli

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type IssueVisibility string

const (
	IssuePublic IssueVisibility = "public"
	IssuePrivate IssueVisibility = "private"
)

func DispatchIssue(command string, args []string) {

	switch command {
		case "list":
			fs := NewCmd("issue list", "Usage: %s <org> <repo> [--pageSize N] [--pageToken PAGE_TOKEN]\n", flag.ContinueOnError)
			
			pageSize := fs.Int("pageSize", 30, "max items to list")
            filter := fs.String("filter", "", "filter")
            sortBy := fs.String("sortBy", "", "sort by")
			pageToken := fs.String("pageToken", "", "page token for pagination")

			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 2, "Usage: issue list <org> <repo> [--pageSize N] [--pageToken PAGE_TOKEN]")
				if rem == nil {
					return
				}

				ListIssues(rem[0], rem[1], int64(*pageSize), *filter, *sortBy, *pageToken)
			}
			
		case "create":
			fs := NewCmd("issue create", "Usage: %s <org> <repo> <title> [--visibility public|private]\n", flag.ContinueOnError)
			
			visibility := fs.String("visibility", "public", "visibility: public|private")

			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 3, "Usage: issue create <org> <repo> <title> [--visibility public|private]")

				vis := IssuePublic
				if *visibility == "private" {
					vis = IssuePrivate
				}

				CreateIssue(rem[0], rem[1], rem[2], vis)
			}
			
			return

		case "update":
			fs := NewCmd("issue update", "Usage: %s <org> <repo> <id> <json>\n", flag.ContinueOnError)
			
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 4, "Usage: issue update <org> <repo> <id> <json>")

				var fields map[string]interface{}

				f, err := ParseJson([]byte(strings.Join(rem[3:], " ")))
				if err == nil {
					fields = f
				}

				UpdateIssue(rem[0], rem[1], rem[2], fields)
			}

		case "view":
			fs := NewCmd("issue view", "Usage: %s <org> <repo> <slug>\n", flag.ContinueOnError)
			
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 3, "Usage: issue view <org> <repo> <id>")
	
				ViewIssue(rem[0], rem[1], rem[2])
			}
			

		case "--help", "-h", "help", "":
			printIssueHelp()

		default:
			printIssueHelp()
    }
}

// printIssueHelp печатает краткую справку по командам issue.
func printIssueHelp() {
    fmt.Fprintln(os.Stderr, `issue commands:
  issue list <org> <repo> [--size N] [--filter STATE] [--token PAGE_TOKEN]
  issue create <org> <repo> <title> [--body BODY] [--visibility public|private]
  issue update <org> <repo> <id> [--body BODY] [--fields JSON]
  issue view <org> <repo> <id> [--json] [--verbose]
Use "issue <command> --help" for command-specific flags.`)
}

func ListIssues(orgSlug, repoSlug string, pageSize int64, filter, sortBy, pageToken string) {
    path := "me/issues"
    if orgSlug != "" && repoSlug != "" {
        path = fmt.Sprintf("repos/%s/%s/issues", orgSlug, repoSlug)
    }

    q := make([]string, 0, 4)
    if pageSize > 0 {
        q = append(q, fmt.Sprintf("page_size=%d", pageSize))
    }
    if pageToken != "" {
        q = append(q, "page_token="+url.QueryEscape(pageToken))
    }
    if sortBy != "" {
        q = append(q, "sort_by="+url.QueryEscape(sortBy))
    }
    if filter != "" {
        q = append(q, "filter="+url.QueryEscape(filter))
    }
    if len(q) > 0 {
        path = path + "?" + strings.Join(q, "&")
    }

    resp, err := DoRequest("GET", path, nil)
    Ensure(err)

    var items []any
    if arr, ok := resp["issues"].([]any); ok {
        items = arr
    } else if arr, ok := resp["data"].([]any); ok {
        items = arr
    } else if arr, ok := resp["items"].([]any); ok {
        items = arr
    } else {
        fmt.Println(ToJson(resp))
        return
    }

    fmt.Printf("Issues  %s/%s\n\n", orgSlug, repoSlug)
    for _, it := range items {
        m, ok := it.(map[string]any)
        if !ok {
            continue
        }
        id := ShortID(fmtString(m["id"], m["slug"]))
        title := fmtString(m["title"])
        state := fmtString(m["status"], m["state"], m["status_slug"])
        assignee := ""
        if a, ok := m["assignee"].(map[string]any); ok {
            assignee = fmtString(a["slug"], a["id"])
        }
        labels := joinLabels(m["labels"])
        due := prettyTimeShortAny(m["deadline"])
        updated := prettyTimeShortAny(m["updated_at"])
        extra := []string{}
        if assignee != "" {
            extra = append(extra, "assignee:"+assignee)
        }
        if labels != "" {
            extra = append(extra, "labels:"+labels)
        }
        if due != "" {
            extra = append(extra, "due:"+due)
        }
        extraStr := ""
        if len(extra) > 0 {
            extraStr = " — " + strings.Join(extra, ", ")
        }
        fmt.Printf("%s %s  %-60s  %s%s\n", id, stateSymbol(state), TruncateString(title, 60), updated, extraStr)
    }
}

func CreateIssue(orgSlug, repoSlug, issueTitle string, visibility IssueVisibility) {
    body := map[string] any{
        "title":      issueTitle,
        "visibility": visibility,
    }
    
    result, err := DoRequest("POST", fmt.Sprintf("repos/%s/%s/issues", orgSlug, repoSlug), body)
    Ensure(err)

    id := ShortID(fmtString4(result["id"])) + "/" + result["slug"]
    title := fmtString(result["title"])
    description := fmtString(result["description"])
    status := fmtString(result["status"], result["state"])
    author := ""
    if a, ok := result["author"].(map[string]any); ok {
        author = fmtString(a["slug"], a["id"], a["name"])
    }
    assignee := ""
    if a, ok := result["assignee"].(map[string]any); ok {
        assignee = fmtString(a["slug"], a["id"], a["name"])
    }
    labels := joinLabels(result["labels"])
    created := prettyTimeShortAny(result["created_at"])
    url := fmtString(result["html_url"], result["url"], result["web_url"])

    fmt.Println()
    fmt.Printf("Issue created: %s\n", id)
    fmt.Printf("  Title     : %s\n", title)
    if description != "" {
        desc := description
        if len(desc) > 200 {
            desc = TruncateString(desc, 200)
        }
        fmt.Printf("  Description: %s\n", desc)
    }
    if url != "" {
        fmt.Printf("  URL       : %s\n", url)
    }
    fmt.Printf("  Repo      : %s/%s\n", orgSlug, repoSlug)
    if author != "" {
        fmt.Printf("  Author    : %s\n", author)
    }
    if assignee != "" {
        fmt.Printf("  Assignee  : %s\n", assignee)
    }
    if labels != "" {
        fmt.Printf("  Labels    : %s\n", labels)
    }
    if status != "" {
        fmt.Printf("  Status    : %s\n", status)
    }
    if created != "" {
        fmt.Printf("  Created   : %s\n", created)
    }
    fmt.Println()
}

func ViewIssue(orgSlug, repoSlug, issueSlug string) {
    path := ""
    if orgSlug != "" && repoSlug != "" {
        path = fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    } else {
        path = fmt.Sprintf("issues/id:%s", issueSlug)
    }
    result, err := DoRequest("GET", path, nil)
    Ensure(err)

    id := fmtString(result["id"], result["slug"])
    title := fmtString(result["title"])
    state := fmtString(result["status"], result["state"], result["status_slug"])
    author := ""
    if a, ok := result["author"].(map[string]any); ok {
        author = fmtString(a["slug"], a["id"])
    }
    assignee := ""
    if a, ok := result["assignee"].(map[string]any); ok {
        assignee = fmtString(a["slug"], a["id"])
    }
    labels := joinLabels(result["labels"])
    created := prettyTimeShortAny(result["created_at"])
    updated := prettyTimeShortAny(result["updated_at"])
    priority := fmtString(result["priority"])
    milestone := ""
    if m, ok := result["milestone"].(map[string]any); ok {
        milestone = fmtString(m["slug"], m["id"])
    }
    deadline := prettyTimeShortAny(result["deadline"])
    description := fmtString(result["description"], result["body"])

    stateTag := strings.ToUpper(state)
    fmt.Printf("%s\n", title)
    fmt.Printf("issue %s  %s\n\n", id, stateTag)

    meta := []string{}
    if author != "" {
        meta = append(meta, "author:"+author)
    }
    if assignee != "" {
        meta = append(meta, "assignee:"+assignee)
    }
    if priority != "" {
        meta = append(meta, "priority:"+priority)
    }
    if milestone != "" {
        meta = append(meta, "milestone:"+milestone)
    }
    if labels != "" {
        meta = append(meta, "labels:"+labels)
    }
    if deadline != "" {
        meta = append(meta, "deadline:"+deadline)
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

    // linked PRs, comments count summary if available
    if lprs, ok := result["linked_prs"].([]any); ok && len(lprs) > 0 {
        fmt.Printf("Linked PRs: %d\n", len(lprs))
    }
    if oc, ok := result["comments_count"]; ok {
        fmt.Printf("Comments: %v\n", oc)
    }
}

func UpdateIssue(orgSlug, repoSlug, issueSlug string, fields map[string] any) {
    if fields == nil {
        fields = make(map[string]any)
    }
    path := fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    result, err := DoRequest("PATCH", path, fields)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func CloseIssue(orgSlug, repoSlug, issueSlug string) {
    body := map[string]any{
        "status_slug": "closed",
    }
    path := fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    result, err := DoRequest("PATCH", path, body)
    Ensure(err)
    fmt.Println(ToJson(result))
}


func stateSymbol(state string) string {
    switch strings.ToLower(state) {
    case "open", "opened":
        return "○"
    case "inprogress", "in_progress", "in progress":
        return "◐"
    case "closed", "done":
        return "●"
    default:
        return "·"
    }
}

func prettyTimeShortAny(v any) string {
    if v == nil {
        return ""
    }
    if s, ok := v.(string); ok && s != "" {
        if t, err := time.Parse(time.RFC3339, s); err == nil {
            return t.Format("2006-01-02")
        }
        return s
    }
    return ""
}

func joinLabels(labels any) string {
    if labels == nil {
        return ""
    }
    switch arr := labels.(type) {
    case []any:
        out := make([]string, 0, len(arr))
        for _, it := range arr {
            if m, ok := it.(map[string]any); ok {
                if n, ok2 := m["name"].(string); ok2 && n != "" {
                    out = append(out, n)
                    continue
                }
                if s, ok2 := m["slug"].(string); ok2 && s != "" {
                    out = append(out, s)
                    continue
                }
            }
            if s, ok := it.(string); ok && s != "" {
                out = append(out, s)
            }
        }
        return strings.Join(out, ", ")
    default:
        return fmt.Sprint(arr)
    }
}


