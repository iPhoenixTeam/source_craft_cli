package cli

import (
    "fmt"
    "time"
)

type MilestoneStatus string

const (
    MilestoneOpen   MilestoneStatus = "open"
    MilestoneClosed MilestoneStatus = "closed"
)

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