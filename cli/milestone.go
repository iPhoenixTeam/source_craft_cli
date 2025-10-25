package cli

import (
	"fmt"
	"strings"
	"time"
)

type MilestoneStatus string

const (
    MilestoneOpen   MilestoneStatus = "open"
    MilestoneClosed MilestoneStatus = "closed"
)

func ExecuteMilestone(command string, args... string) {
    switch command {
		case "list":
			ListRepo(args[3])
		case "create":
			CreateRepo(args[3], args[4], args[4], "", RepoPublic, false)
		case "fork":
			ForkRepo(args[3], args[4], args[5], true)
		case "view":
			ViewRepo(args[3], args[4])
		default:
			//help
	}
}

func ListMilestones(orgSlug, repoSlug string, pageSize int, pageToken string) {
    path := fmt.Sprintf("repos/%s/%s/milestones", orgSlug, repoSlug)
    q := make(map[string]any)
    if pageSize > 0 {
        q["page_size"] = pageSize
    }
    if pageToken != "" {
        q["page_token"] = pageToken
    }
    result, err := Execute1("GET", path, q)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func CreateMilestone(orgSlug, repoSlug, name, description string, startDate, deadline time.Time, status MilestoneStatus) {
    body := map[string]any{
        "name":        name,
        "description": description,
    }
    if !startDate.IsZero() {
        body["start_date"] = startDate.Format(time.RFC3339)
    }
    if !deadline.IsZero() {
        body["deadline"] = deadline.Format(time.RFC3339)
    }
    if status != "" {
        body["status"] = string(status)
    }

    path := fmt.Sprintf("repos/%s/%s/milestones", orgSlug, repoSlug)
    result, err := Execute1("POST", path, body)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func ViewMilestoneByID(milestoneID string) {
    path := fmt.Sprintf("milestones/id:%s", milestoneID)
    result, err := Execute1("GET", path, nil)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func ViewMilestone(orgSlug, repoSlug, milestoneSlug string) {
    path := fmt.Sprintf("repos/%s/%s/milestones/%s", orgSlug, repoSlug, milestoneSlug)
    result, err := Execute1("GET", path, nil)
    Ensure(err)
    fmt.Println(ToJson(result))
}


func printMilestoneLine(id, title, state, due string) {
    // id (short) | state | due | title
    shortID := id
    if len(id) > 8 {
        shortID = id[:8]
    }
    stateSym := stateSymbol1(state)
    dueStr := due
    if dueStr == "" {
        dueStr = "no due date"
    }
    fmt.Printf("%s %s  %-10s  %s\n", shortID, stateSym, dueStr, title)
}

func stateSymbol1(state string) string {
    switch strings.ToLower(state) {
    case "open", "opened":
        return "○"
    case "closed", "done":
        return "●"
    default:
        return "·"
    }
}

// ListMilestonesPretty делает запрос и печатает аккуратно отформатированный список
func ListMilestonesPretty(orgSlug, repoSlug string) {
    path := fmt.Sprintf("repos/%s/%s/milestones", orgSlug, repoSlug)
    raw, err := Execute1("GET", path, nil)
    Ensure(err)

    // Попробуем извлечь массив из разных форматов ответа
    var items []any
    if arr, ok := raw["items"].([]any); ok && len(arr) > 0 {
        items = arr
    } else if data, ok := raw["data"].([]any); ok && len(data) > 0 {
        items = data
    } else if arrAll, ok := raw["milestones"].([]any); ok && len(arrAll) > 0 {
        items = arrAll
    } else {

        fmt.Println(ToJson(raw))
        return
    }

    fmt.Printf("Milestones for %s/%s\n\n", orgSlug, repoSlug)

    // Вывод строк
    for _, it := range items {
        m, ok := it.(map[string]any)
        if !ok {
            continue
        }
        id := fmtString4(m["id"])
        title := fmtString4(m["title"])
        state := fmtString4(m["status"], m["state"], m["status_slug"])
        due := parseDateFromMap(m, "deadline", "due_date", "due")
        printMilestoneLine(id, title, state, due)
    }
}

func ViewMilestonePretty(orgSlug, repoSlug, milestoneSlug string) {
    path := fmt.Sprintf("repos/%s/%s/milestones/%s", orgSlug, repoSlug, milestoneSlug)
    result, err := Execute1("GET", path, nil)
    Ensure(err)

    // поля
    id := fmtString4(result["id"])
    title := fmtString4(result["title"])
    description := fmtString4(result["description"], result["body"])
    state := fmtString4(result["status"], result["state"], result["status_slug"])
    created := fmtString4(result["created_at"], result["created"])
    updated := fmtString4(result["updated_at"], result["updated"])
    due := parseDateFromMap(result, "deadline", "due_date", "due")

    // Header (title line + meta)
    stateTag := strings.ToUpper(state)
    fmt.Printf("%s (%s)\n", title, stateTag)
    fmt.Printf("milestone %s\n\n", id)

    if description != "" {
        fmt.Println(indentMultiline(description, 2))
        fmt.Println()
    }

    fmt.Printf("Created: %s\n", prettyTime(created))
    if updated != "" && updated != created {
        fmt.Printf("Updated: %s\n", prettyTime(updated))
    }
    if due != "" {
        fmt.Printf("Due: %s\n", due)
    }
    // Сводка: issues counts
    if openCount, ok := numberFrom(result["open_issues_count"]); ok {
        if closedCount, ok2 := numberFrom(result["closed_issues_count"]); ok2 {
            fmt.Printf("Issues: %d open, %d closed\n", openCount, closedCount)
        } else {
            fmt.Printf("Issues open: %d\n", openCount)
        }
    }
}

// Вспомогательные функции

func fmtString4(vals ...any) string {
    for _, v := range vals {
        if v == nil {
            continue
        }
        if s, ok := v.(string); ok && s != "" {
            return s
        }
    }
    return ""
}

func parseDateFromMap(m map[string]any, keys ...string) string {
    for _, k := range keys {
        if v, ok := m[k]; ok && v != nil {
            if s, ok := v.(string); ok && s != "" {
                // попытка привести к читаемому формату
                if t, err := time.Parse(time.RFC3339, s); err == nil {
                    return t.Format("2006-01-02")
                }
                return s
            }
        }
    }
    return ""
}

func prettyTime(s string) string {
    if s == "" {
        return ""
    }
    if t, err := time.Parse(time.RFC3339, s); err == nil {
        // "Jan 02 2006" like git
        return t.Format("Jan 02 2006")
    }
    return s
}

func indentMultiline(s string, indent int) string {
    pad := strings.Repeat(" ", indent)
    lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
    for i := range lines {
        lines[i] = pad + lines[i]
    }
    return strings.Join(lines, "\n")
}

func numberFrom(v any) (int, bool) {
    switch n := v.(type) {
    case int:
        return n, true
    case int64:
        return int(n), true
    case float64:
        return int(n), true
    case float32:
        return int(n), true
    default:
        return 0, false
    }
}