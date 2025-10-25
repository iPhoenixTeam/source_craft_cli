package cli

import (
	"fmt"
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
			requireArgs(args, 2, "")
			ListIssues(args[0], args[1])
		case "create":
			requireArgs(args, 3, "")
			CreateRepo(args[0], args[1], args[1], "", RepoPublic, false)
		case "fork":
			requireArgs(args, 3, "")
			ForkRepo(args[0], args[1], args[2], true)
		case "view":
			requireArgs(args, 2, "")
			ViewRepo(args[0], args[1])
		default:
			//help
	}
}

func ListIssues(org_slug, repo_slug string) {
    result, err := Execute1("GET", fmt.Sprintf("/repos/%s/%s/issues", org_slug, repo_slug), nil)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func CreateIssue(orgSlug, repoSlug, issueTitle string, visibility IssueVisibility, createReadme bool) {
    body := map[string] any{
        "title":      issueTitle,
        "visibility": visibility,
    }
    if createReadme {
        body["properties"] = map[string]any{
            "create_readme": true,
        }
    }
    path := fmt.Sprintf("repos/%s/%s/issues", orgSlug, repoSlug)
    result, err := Execute1("POST", path, body)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func ViewIssue(orgSlug, repoSlug, issueSlug string) {
    path := fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    result, err := Execute1("GET", path, nil)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func UpdateIssue(orgSlug, repoSlug, issueSlug string, fields map[string] any) {
    if fields == nil {
        fields = make(map[string]any)
    }
    path := fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    result, err := Execute1("PATCH", path, fields)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func CloseIssue(orgSlug, repoSlug, issueSlug string) {
    body := map[string]any{
        "status_slug": "closed",
    }
    path := fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    result, err := Execute1("PATCH", path, body)
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


// ListIssuesPretty prints issues in a concise git-like list
func ListIssuesPretty(orgSlug, repoSlug string, pageSize int, pageToken string) {
    path := ""
    if orgSlug != "" && repoSlug != "" {
        path = fmt.Sprintf("repos/%s/%s/issues", orgSlug, repoSlug)
    } else {
        path = "me/issues"
    }
    q := make(map[string]any)
    if pageSize > 0 {
        q["page_size"] = pageSize
    }
    if pageToken != "" {
        q["page_token"] = pageToken
    }
    resp, err := Execute1("GET", path, q)
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

// ViewIssuePretty prints a single issue in a detailed git-issue style
func ViewIssuePretty(orgSlug, repoSlug, issueSlug string) {
    path := ""
    if orgSlug != "" && repoSlug != "" {
        path = fmt.Sprintf("repos/%s/%s/issues/%s", orgSlug, repoSlug, issueSlug)
    } else {
        path = fmt.Sprintf("issues/id:%s", issueSlug)
    }
    result, err := Execute1("GET", path, nil)
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