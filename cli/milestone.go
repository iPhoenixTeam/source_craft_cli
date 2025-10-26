package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type MilestoneStatus string

const (
    MilestoneOpen   MilestoneStatus = "open"
    MilestoneClosed MilestoneStatus = "closed"
)

func printMilestoneHelp() {
    fmt.Fprintln(os.Stderr, `milestone commands:
  milestone list <org> <repo> [--size N] [--token PAGE_TOKEN]
  milestone create <org> <repo> <title>
  milestone view <org> <repo> <id>
Use "milestone <command> --help" for command-specific flags.`)
}

func DispatchMilestone(command string, args []string) {
    switch command {

        case "list":
            fs := NewCmd("milestone list", "Usage: %s list <org> <repo>\n", flag.ContinueOnError)
            
            if err := fs.Parse(args); err == nil {
                rem := Require(fs, 2, "Usage: milestone list <org> <repo>")

                ListMilestone(rem[0], rem[1])
            }

        case "create":
            fs := NewCmd("milestone create", "Usage: %s create <org> <repo> <title>\n", flag.ContinueOnError)
            
            if err := fs.Parse(args); err == nil {
                rem := Require(fs, 3, "Usage: milestone create <org> <repo> <title>")
                
                CreateMilestone(rem[0], rem[1], rem[2])
            }

        case "view":
            fs := NewCmd("milestone view", "Usage: %s view <org> <repo> <id>\n", flag.ContinueOnError)
            
            if err := fs.Parse(args); err == nil {
                rem := Require(fs, 3, "Usage: milestone view <org> <repo> <id>")
                
                ViewMilestone(rem[0], rem[1], rem[2])
            }

        case "--help", "-h", "help", "":
            printMilestoneHelp()

        default:
            printMilestoneHelp()
    }
}

func ListMilestone(orgSlug, repoSlug string) {
    path := fmt.Sprintf("/repos/%s/%s/milestones", orgSlug, repoSlug)
    raw, err := DoRequest("GET", path, nil)
    Ensure(err)

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

    for _, it := range items {
        m, ok := it.(map[string]any)
        if !ok {
            continue
        }
        id := ToString(m["id"])
        title := ToString(m["title"])
        state := ToString(m["status"])
        due := ParseDateFromMap(m, "deadline", "due_date", "due")
        printMilestoneLine(id, title, state, due)
    }
}


func CreateMilestone(orgSlug, repoSlug, milestoneName string) {
    body := map[string] any {
        "name":        milestoneName,
    }

    result, err := DoRequest("POST", fmt.Sprintf("/repos/%s/%s/milestones", orgSlug, repoSlug), body)
    Ensure(err)
    fmt.Println(ToJson(result))
}

func ViewMilestone(orgSlug, repoSlug, milestoneSlug string) {
    result, err := DoRequest("GET", fmt.Sprintf("/repos/%s/%s/milestones/%s", orgSlug, repoSlug, milestoneSlug), nil)
    Ensure(err)

    id := ToString(result["id"])
    title := ToString(result["title"])
    description := ToString(result["description"])
    state := ToString(result["status"])
    created := ToString(result["created_at"])
    updated := ToString(result["updated_at"])
    due := ParseDateFromMap(result, "deadline", "due_date", "due")

    stateTag := strings.ToUpper(state)
    fmt.Printf("%s (%s)\n", title, stateTag)
    fmt.Printf("milestone %s\n\n", id)

    if description != "" {
        fmt.Println(IndentMultilineString(description, 2))
        fmt.Println()
    }

    fmt.Printf("Created: %s\n", prettyTime(created))
    if updated != "" && updated != created {
        fmt.Printf("Updated: %s\n", prettyTime(updated))
    }
    if due != "" {
        fmt.Printf("Due: %s\n", due)
    }

    if openCount, ok := NumberFrom(result["open_issues_count"]); ok {
        if closedCount, ok2 := NumberFrom(result["closed_issues_count"]); ok2 {
            fmt.Printf("Issues: %d open, %d closed\n", openCount, closedCount)
        } else {
            fmt.Printf("Issues open: %d\n", openCount)
        }
    }
}

func printMilestoneLine(id, title, state, due string) {
    shortID := ShortID(id)
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
        return "○ open(ed)"
    case "closed", "done":
        return "● done"
    default:
        return "·"
    }
}