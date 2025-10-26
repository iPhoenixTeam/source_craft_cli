package cli

import (
	"flag"
	"fmt"
	"strings"
)

func DispatchPolicy(command string, args... string) {
	switch command {
		case "view":
			fs := NewCmd("repo view", "Usage: %s list <org> <repo> [options]\n", flag.ContinueOnError)
        
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 2, "Usage: %s list <org> <repo>")
				
				PolicyView(rem[0], rem[1])
			}
		case "update":
			fs := NewCmd("repo update", "Usage: %s list <org> <repo> [options]\n", flag.ContinueOnError)
        
			if err := fs.Parse(args); err == nil {
				rem := Require(fs, 2, "Usage: %s list <org> <repo>")
				
				PolicyView(rem[0], rem[1])
			}
			
		default:
			//help
	}
}

func DispatchReviewRules(command string, args... string) {
	switch command {
		case "view":
			//requireArgs(args, 2, "")
			PolicyView(args[0], args[1])
		default:
			//help
	}
}

func PolicyView(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/branch_policies", orgSlug, repoSlug)
	result, err := DoRequest("GET", path, nil)
	Ensure(err)
	printPolicyList(result)
}

func PolicyUpdate(orgSlug, repoSlug, branch string, rules map[string]any) {
	if rules["branch"] != "" {rules["branch"] = branch}
	
	path := fmt.Sprintf("repos/%s/%s/branch_policies", orgSlug, repoSlug)
	result, err := DoRequest("PUT", path, rules)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func ReviewRulesView(orgSlug, repoSlug string) {
	path := fmt.Sprintf("repos/%s/%s/review_rules", orgSlug, repoSlug)
	result, err := DoRequest("GET", path, nil)
	Ensure(err)
	printReviewRulesList(result)
}

func ReviewRulesUpdate(orgSlug, repoSlug, ruleID string, payload map[string]any) {
	path := fmt.Sprintf("repos/%s/%s/review_rules/%s", orgSlug, repoSlug, ruleID)
	result, err := DoRequest("PATCH", path, payload)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func printPolicyList(m map[string]any) {
	repo := ToString(m["slug"])
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
		branch := ToString(p["branch"])
		enforced := ToString(p["enforced"])
		providers := joinStringsFrom(p["providers"])
		protectors := joinStringsFrom(p["protectors"])
		updated := prettyTime(p["last_updated"])

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
	repo := ToString(m["id"])
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
		id := ToString(r["id"])
		name := ToString(r["name"])
		enabled := ToString(r["enabled"])
		conditions := summarizeConditions(r["conditions"])
		requirements := summarizeRequirements(r["requirements"])
		updated := prettyTime(r["last_updated"])

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

func extractArrayCandidates(m JsonObject, keys ...string) []any {
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
				if name := ToString(mm["id"]); name != "" {
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
				parts = append(parts, ToString(mm["type"]), ToString(mm["field"]), ToString(mm["pattern"]))
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
				parts = append(parts, ToString(mm["type"]), ToString(mm["requirement"]), ToString(mm["name"]))
			} else {
				parts = append(parts, fmt.Sprint(it))
			}
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}
