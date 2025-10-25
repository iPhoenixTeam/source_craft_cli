package cli

import (
    "fmt"
    "regexp"
    "strings"
    "time"
)

func CodeSearch(query, orgSlug, repoSlug string, pageSize int, pageToken string) {
    paths := candidateCodeSearchPaths(orgSlug, repoSlug)
    var resp map[string]any
    var err error
    var tried []string
    for _, p := range paths {
        tried = append(tried, p)
        q := make(map[string]any)
        if query != "" {
            q["q"] = query
        }
        if pageSize > 0 {
            q["page_size"] = pageSize
        }
        if pageToken != "" {
            q["page_token"] = pageToken
        }
        resp, err = Execute1("GET", p, q)
        if err == nil {
            break
        }
    }
    Ensure(err)

    items := extractCodeItems(resp)
    if len(items) == 0 {
        fmt.Printf("No results for %q\n", query)
        return
    }

    fmt.Printf("Code search results for %q\n\n", query)
    for _, it := range items {
        printCodeHit(it, query)
    }
}

// candidate endpoints to try (tries repo-scoped first if repoSlug provided)
func candidateCodeSearchPaths(orgSlug, repoSlug string) []string {
    if orgSlug != "" && repoSlug != "" {
        return []string{
            fmt.Sprintf("repos/%s/%s/search/code", orgSlug, repoSlug),
            fmt.Sprintf("repos/%s/%s/code/search", orgSlug, repoSlug),
            fmt.Sprintf("repos/%s/%s/search", orgSlug, repoSlug),
        }
    }
    if orgSlug != "" && repoSlug == "" {
        return []string{
            fmt.Sprintf("orgs/%s/search/code", orgSlug),
            fmt.Sprintf("orgs/%s/code/search", orgSlug),
        }
    }
    return []string{
        "search/code",
        "code/search",
        "search",
    }
}

// codeHit is normalized structure for printing
type codeHit struct {
    Repo        string
    FilePath    string
    Line        int
    Preview     string
    Language    string
    CommitID    string
    UpdatedAt   string
    Score       float64
    ContextLeft string
    ContextRight string
}

func extractCodeItems(resp map[string]any) []codeHit {
    out := []codeHit{}
    // try several common shapes: "items", "results", "files", "hits"
    var arr []any
    if a, ok := resp["items"].([]any); ok {
        arr = a
    } else if a, ok := resp["results"].([]any); ok {
        arr = a
    } else if a, ok := resp["files"].([]any); ok {
        arr = a
    } else if a, ok := resp["hits"].([]any); ok {
        arr = a
    } else {
        // sometimes API returns top-level array encoded under other keys
        // try to coerce resp itself if it looks like a file result (best-effort)
        return tryCoerceSingleResult(resp)
    }

    for _, it := range arr {
        if m, ok := it.(map[string]any); ok {
            h := codeHit{
                Repo:      fmtString2(m["repo"], m["repository"], m["repo_slug"]),
                FilePath:  fmtString2(m["path"], m["file"], m["file_path"]),
                Language:  fmtString2(m["language"]),
                CommitID:  fmtString2(m["commit"], m["commit_id"]),
                UpdatedAt: fmtString2(m["updated_at"], m["last_updated"]),
                Score:     float64From(m["score"]),
            }
            // preview/snippet fields
            h.Preview = fmtString2(m["preview"], m["snippet"], m["line_preview"], m["context"])
            if ln, ok := numberFromAny(m["line"]); ok {
                h.Line = ln
            } else if ln2, ok := numberFromAny(m["line_number"]); ok {
                h.Line = ln2
            }
            // context
            h.ContextLeft = fmtString2(m["context_left"], m["before"])
            h.ContextRight = fmtString2(m["context_right"], m["after"])
            out = append(out, h)
        }
    }
    return out
}

func tryCoerceSingleResult(m map[string]any) []codeHit {
    // if response itself contains file-like keys, return single item
    if _, hasPath := m["path"]; hasPath || m["file"] != nil {
        h := codeHit{
            Repo:      fmtString2(m["repo"], m["repository"]),
            FilePath:  fmtString2(m["path"], m["file"]),
            Preview:   fmtString2(m["preview"], m["snippet"], m["content"]),
            Language:  fmtString2(m["language"]),
            CommitID:  fmtString2(m["commit"], m["commit_id"]),
            UpdatedAt: fmtString2(m["updated_at"], m["last_updated"]),
            Score:     float64From(m["score"]),
        }
        if ln, ok := numberFromAny(m["line"]); ok {
            h.Line = ln
        }
        return []codeHit{h}
    }
    return nil
}

func printCodeHit(h codeHit, query string) {
    // header: repo/path:line  (language) [score]
    loc := h.FilePath
    if loc == "" {
        loc = "(unknown path)"
    }
    lineStr := ""
    if h.Line > 0 {
        lineStr = fmt.Sprintf(":%d", h.Line)
    }
    scoreStr := ""
    if h.Score > 0 {
        scoreStr = fmt.Sprintf("  score: %.2f", h.Score)
    }
    lang := ""
    if h.Language != "" {
        lang = fmt.Sprintf(" (%s)", h.Language)
    }
    fmt.Printf("%s/%s%s%s\n", h.Repo, loc, lineStr, lang)
    if h.CommitID != "" {
        fmt.Printf("  commit: %s", ShortID(h.CommitID))
    }
    if h.UpdatedAt != "" {
        fmt.Printf("  updated: %s", prettyTimeShortAny(h.UpdatedAt))
    }
    if scoreStr != "" {
        fmt.Printf(" %s", scoreStr)
    }
    fmt.Println()

    // print snippet with highlighted query (simple case-insensitive)
    if h.Preview != "" {
        printSnippetWithHighlight(h.Preview, query)
        fmt.Println()
        return
    }

    // fallback: print context if available
    if h.ContextLeft != "" || h.ContextRight != "" {
        ctx := strings.TrimSpace(h.ContextLeft + " " + h.ContextRight)
        if ctx != "" {
            printSnippetWithHighlight(ctx, query)
            fmt.Println()
            return
        }
    }

    fmt.Println("  (no preview available)\n")
}

func printSnippetWithHighlight(s, query string) {
    lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
    // limit lines to reasonable number
    max := 8
    if len(lines) > max {
        lines = lines[:max]
    }
    q := strings.TrimSpace(query)
    re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(q))
    for i, ln := range lines {
        // show line numbers relative to snippet
        prefix := fmt.Sprintf("  %2d | ", i+1)
        // highlight occurrences by uppercasing matched fragment (minimal approach)
        lnH := re.ReplaceAllStringFunc(ln, func(m string) string {
            return strings.ToUpper(m)
        })
        fmt.Println(prefix + TruncateString(lnH, 200))
    }
}

func fmtString2(vals ...any) string {
    for _, v := range vals {
        if v == nil {
            continue
        }
        switch t := v.(type) {
        case string:
            if t != "" {
                return t
            }
        case map[string]any:
            if s, ok := t["slug"].(string); ok && s != "" {
                return s
            }
        }
    }
    return ""
}

func numberFromAny(v any) (int, bool) {
    switch n := v.(type) {
    case int:
        return n, true
    case int64:
        return int(n), true
    case float64:
        return int(n), true
    case float32:
        return int(n), true
    case string:
        var x int
        if _, err := fmt.Sscanf(n, "%d", &x); err == nil {
            return x, true
        }
    }
    return 0, false
}

func float64From(v any) float64 {
    switch t := v.(type) {
    case float64:
        return t
    case float32:
        return float64(t)
    case int:
        return float64(t)
    case int64:
        return float64(t)
    case string:
        var f float64
        fmt.Sscanf(t, "%f", &f)
        return f
    default:
        return 0
    }
}

func prettyTimeShortAny(val any) string {
    if val == nil {
        return ""
    }
    if s, ok := val.(string); ok && s != "" {
        if t, err := time.Parse(time.RFC3339, s); err == nil {
            return t.Format("2006-01-02")
        }
        return s
    }
    return ""
}


