package cli

import (
	"fmt"
	"strings"
)

func DispatchPolicy(command string, args... string) {
	switch command {
		case "list":
			requireArgs(args, 1, "")
			ListRepo(args[3])
		case "create":
			requireArgs(args, 3, "")
			CreateRepo(args[3], args[4], args[4], "", RepoPublic, false)
		case "fork":
			requireArgs(args, 3, "")
			ForkRepo(args[3], args[4], args[5], true)
		case "view":
			requireArgs(args, 2, "")
			ViewRepo(args[3], args[4])
		default:
			//help
	}
}

func PolicyView(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/branch_policies", orgSlug, repoSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)
	printPolicyList(result)
}

func PolicyUpdate(orgSlug, repoSlug, branch string, rules map[string]any) {
	rules["branch"] = branch;
	path := fmt.Sprintf("repos/%s/%s/branch_policies", orgSlug, repoSlug)
	result, err := Execute1("PUT", path, rules)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func ReviewRulesView(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/review_rules", orgSlug, repoSlug)
	result, err := Execute1("GET", path, nil)
	Ensure(err)
	printReviewRulesList(result)
}

func ReviewRulesUpdate(orgSlug, repoSlug, ruleID string, payload map[string]any) {
	path := fmt.Sprintf("repos/%s/%s/review_rules/%s", orgSlug, repoSlug, ruleID)
	result, err := Execute1("PATCH", path, payload)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func printPolicyList(m map[string]any) {
	repo := fmtString(m["repo"], m["slug"], m["id"])
	fmt.Printf("Branch policies for %s\n\n", repo)

	items := extractArrayCandidates(m, "policies", "items", "data", "branch_policies")
	if len(items) == 0 {
		fmt.Println("(no policies)")
		return
	}

	for _, it := range items {
		p, ok := it.(map[string]any)
		if !ok {
			continue
		}
		branch := fmtString(p["branch"], p["branch_name"], p["pattern"])
		enforced := fmtString(p["enforced"], p["enabled"])
		providers := joinStringsFrom(p["providers"])
		protectors := joinStringsFrom(p["protectors"])
		updated := prettyTimeShortAny(p["last_updated"])

		fmt.Printf("branch: %s\n", branch)
		fmt.Printf("  enforced: %s  updated: %s\n", enforced, updated)
		if providers != "" {
			fmt.Printf("  providers: %s\n", providers)
		}
		if protectors != "" {
			fmt.Printf("  rules: %s\n", protectors)
		}
		// show raw settings if available
		if cfg, ok := p["config"].(map[string]any); ok && len(cfg) > 0 {
			result, _ := ToJson(cfg)
			fmt.Printf("  config: %s\n", TruncateString(result, 200))
		}
		fmt.Println()
	}
}

func printReviewRulesList(m map[string]any) {
	repo := fmtString(m["repo"], m["slug"], m["id"])
	fmt.Printf("Review rules for %s\n\n", repo)

	items := extractArrayCandidates(m, "rules", "review_rules", "items", "data")
	if len(items) == 0 {
		fmt.Println("(no review rules)")
		return
	}

	for _, it := range items {
		r, ok := it.(map[string]any)
		if !ok {
			continue
		}
		id := fmtString(r["id"], r["slug"])
		name := fmtString(r["name"], r["title"])
		enabled := fmtString(r["enabled"], r["active"])
		conditions := summarizeConditions(r["conditions"])
		requirements := summarizeRequirements(r["requirements"])
		updated := prettyTimeShortAny(r["last_updated"])

		fmt.Printf("%s  %s\n", id, name)
		fmt.Printf("  enabled: %s  updated: %s\n", enabled, updated)
		if conditions != "" {
			fmt.Printf("  conditions: %s\n", conditions)
		}
		if requirements != "" {
			fmt.Printf("  requirements: %s\n", requirements)
		}
		// print raw payload preview when available
		if cfg, _ := ToJson(r); cfg != "" {
			fmt.Printf("  preview: %s\n\n", TruncateString(cfg, 240))
		} else {
			fmt.Println()
		}
	}
}

/* --- small utilities --- */

func extractArrayCandidates(m map[string]any, keys ...string) []any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if arr, ok := v.([]any); ok {
				return arr
			}
		}
	}
	return nil
}

func joinStringsFrom(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []any:
		out := []string{}
		for _, it := range t {
			if s, ok := it.(string); ok && s != "" {
				out = append(out, s)
				continue
			}
			if mm, ok := it.(map[string]any); ok {
				if name := fmtString(mm["name"], mm["slug"], mm["id"]); name != "" {
					out = append(out, name)
				}
			}
		}
		return strings.Join(out, ", ")
	case []string:
		return strings.Join(t, ", ")
	default:
		return fmt.Sprint(v)
	}
}

func summarizeConditions(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []any:
		parts := []string{}
		for _, it := range t {
			if mm, ok := it.(map[string]any); ok {
				parts = append(parts, fmtString(mm["type"], mm["field"], mm["pattern"]))
			} else {
				parts = append(parts, fmt.Sprint(it))
			}
		}
		return strings.Join(parts, "; ")
	case map[string]any:
		res, _ := ToJson(t)
		return TruncateString(res, 200)
	default:
		return fmt.Sprint(v)
	}
}

func summarizeRequirements(v any) string {
	if v == nil {
		return ""
	}
	if arr, ok := v.([]any); ok {
		parts := []string{}
		for _, it := range arr {
			if mm, ok := it.(map[string]any); ok {
				parts = append(parts, fmtString(mm["type"], mm["requirement"], mm["name"]))
			} else {
				parts = append(parts, fmt.Sprint(it))
			}
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}

/* --- reused helpers (assume they already exist elsewhere in package) --- */

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
			if s, ok := t["name"].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
