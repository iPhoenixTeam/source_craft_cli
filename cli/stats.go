package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func StatsRepo(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/stats", orgSlug, repoSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)

	printRepoStats(result)
}

func StatsUser(userSlug string) {
	path := fmt.Sprintf("users/%s/stats", userSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)

	printUserStats(result)
}

func StatsSecurity(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/security/stats", orgSlug, repoSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)

	printSecurityStats(result)
}

func ReportSecurity(orgSlug, repoSlug string, topN int) {
	path := fmt.Sprintf("repos/%s/%s/security/report", orgSlug, repoSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)

	printSecurityReport(result, topN)
}

/* --- Pretty printers --- */

func printRepoStats(m map[string]any) {
	name := fmtString(m["name"], m["slug"])
	lang := fmtString(m["language"])
	created := prettyTimeShortAny(m["created_at"], m["created"])
	updated := prettyTimeShortAny(m["last_updated"], m["updated_at"])

	fmt.Printf("%s\n", name)
	fmt.Printf("repo %s\n\n", fmtString(m["id"], m["slug"]))
	if d := fmtString(m["description"]); d != "" {
		fmt.Println(IndentString(d, 2))
		fmt.Println()
	}
	fmt.Printf("Language:   %s\n", lang)
	fmt.Printf("Created:    %s\n", created)
	fmt.Printf("Updated:    %s\n", updated)
	fmt.Println()

	if counters, ok := m["counters"].(map[string]any); ok {
		fmt.Printf("Forks: %s  PRs: %s  Issues: %s  Branches: %s\n",
			fmtString(counters["forks"]), fmtString(counters["pull_requests"]), fmtString(counters["issues"]), fmtString(counters["branches"]))
	}

	if traffic, ok := m["traffic"].(map[string]any); ok {
		fmt.Println()
		fmt.Println("Traffic")
		printKvLine("Views", traffic["views"])
		printKvLine("Clones", traffic["clones"])
		printKvLine("Unique visitors", traffic["unique_visitors"])
	}

	if activity, ok := m["activity"].(map[string]any); ok {
		fmt.Println()
		fmt.Println("Activity (last period)")
		for k, v := range activity {
			fmt.Printf("  %-18s %v\n", prettyKey(k), v)
		}
	}
}

func printUserStats(m map[string]any) {
	user := fmtString(m["user"], m["slug"], m["id"])
	fmt.Printf("User: %s\n\n", user)

	if totals, ok := m["totals"].(map[string]any); ok {
		printKvLine("Repositories", totals["repos_count"])
		printKvLine("Commits", totals["commits_count"])
		printKvLine("Pull requests", totals["pull_requests_count"])
		printKvLine("Issues", totals["issues_count"])
		fmt.Println()
	}

	if byRepo, ok := m["by_repo"].([]any); ok && len(byRepo) > 0 {
		fmt.Println("Top repositories by activity:")
		entries := make([]string, 0, len(byRepo))
		for _, it := range byRepo {
			if mm, ok := it.(map[string]any); ok {
				n := fmtString(mm["repo_name"], mm["repo_slug"])
				score := fmtString(mm["score"], mm["activity_score"])
				entries = append(entries, fmt.Sprintf("  %-30s %6s", TruncateString(n, 30), score))
			}
		}
		for _, e := range entries {
			fmt.Println(e)
		}
	}
}

func printSecurityStats(m map[string]any) {
	fmt.Printf("Security stats for %s\n\n", fmtString(m["repo"], m["slug"], m["id"]))

	if counts, ok := m["counts"].(map[string]any); ok {
		printKvLine("Critical", counts["critical"])
		printKvLine("High", counts["high"])
		printKvLine("Medium", counts["medium"])
		printKvLine("Low", counts["low"])
		fmt.Println()
	}

	if scanners, ok := m["scanners"].([]any); ok && len(scanners) > 0 {
		fmt.Println("Scanners")
		for _, it := range scanners {
			if mm, ok := it.(map[string]any); ok {
				fmt.Printf("  %-20s  last_scan: %s  issues: %s\n",
					fmtString(mm["name"]), prettyTimeShortAny(mm["last_scan"]), fmtString(mm["issues_count"]))
			}
		}
	}
}

func printSecurityReport(m map[string]any, topN int) {
	fmt.Printf("Security report for %s\n\n", fmtString(m["repo"], m["slug"], m["id"]))

	vulns := collectVulnerabilities(m)
	if len(vulns) == 0 {
		fmt.Println("No findings")
		return
	}

	sort.Slice(vulns, func(i, j int) bool {
		return vulnScore(vulns[i]) > vulnScore(vulns[j])
	})
	if topN <= 0 || topN > len(vulns) {
		topN = len(vulns)
	}

	fmt.Printf("Top %d prioritized risks\n\n", topN)
	for i := 0; i < topN; i++ {
		v := vulns[i]
		fmt.Printf("%2d) [%s] %s\n", i+1, strings.ToUpper(v.Severity), v.Title)
		if v.Package != "" {
			fmt.Printf("     pkg: %s  version: %s\n", v.Package, v.Version)
		}
		if v.Description != "" {
			fmt.Println(IndentString(TruncateString(v.Description, 200), 6))
		}
		fmt.Printf("     score: %.2f  fix_available: %v  references: %d\n\n", v.Score, v.FixAvailable, len(v.References))
	}
}

/* --- helpers for security report --- */

type vulnerability struct {
	Title        string
	Severity     string
	Package      string
	Version      string
	Description  string
	Score        float64
	FixAvailable bool
	References   []string
}

func collectVulnerabilities(m map[string]any) []vulnerability {
	out := []vulnerability{}
	if items, ok := m["vulnerabilities"].([]any); ok {
		for _, it := range items {
			if mm, ok := it.(map[string]any); ok {
				v := vulnerability{
					Title:       fmtString(mm["title"], mm["name"]),
					Severity:    fmtString(mm["severity"], mm["level"]),
					Package:     fmtString(mm["package"], mm["pkg"]),
					Version:     fmtString(mm["version"]),
					Description: fmtString(mm["description"]),
					Score:       float64From(mm["score"]),
					FixAvailable: func() bool {
						if b, ok := mm["fix_available"].(bool); ok {
							return b
						}
						if s := fmtString(mm["fix_version"]); s != "" {
							return true
						}
						return false
					}(),
				}
				if refs, ok := mm["references"].([]any); ok {
					for _, r := range refs {
						v.References = append(v.References, fmtString(r))
					}
				}
				out = append(out, v)
			}
		}
	}
	return out
}

func vulnScore(v vulnerability) float64 {
	// basic heuristic: severity weight + score
	weight := 0.0
	switch strings.ToLower(v.Severity) {
	case "critical":
		weight = 5
	case "high":
		weight = 3
	case "medium":
		weight = 1.5
	case "low":
		weight = 0.5
	default:
		weight = 1
	}
	return weight*10 + v.Score
}

/* --- small utility helpers reused from other files --- */

func printKvLine(k string, v any) {
	fmt.Printf("  %-18s %v\n", k+":", v)
}

func fmtString(vals ...any) string {
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

func prettyKey(k string) string {
	return strings.ReplaceAll(k, "_", " ")
}