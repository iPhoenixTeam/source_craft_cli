package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepoVisibility string

const (
	RepoPublic   RepoVisibility = "public"
	RepoInternal RepoVisibility = "internal"
	RepoPrivate  RepoVisibility = "private"
)

func ExecuteRepo(command string, args... string) {

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

func ListRepo(orgSlug string) {
	result, err := Execute1("GET", fmt.Sprintf("orgs/%s/repos", orgSlug), nil)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func CreateRepo(orgSlug, repoName, repoSlug, description string, visibility RepoVisibility, createReadme bool) {
	body := map[string] any {
		"name":        repoName,
		"slug":        repoSlug,
		"description": description,
		"visibility":  string(visibility),
		"init_settings": map[string]any{
			"default_branch": "",
			"create_readme":  createReadme,
		},
	}
	result, err := Execute1("POST", fmt.Sprintf("orgs/%s/repos", orgSlug), body)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func ViewRepo(orgSlug, repoSlug string) {
	result, err := Execute1("GET", fmt.Sprintf("repos/%s/%s", orgSlug, repoSlug), nil)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func ForkRepo(orgSlug, oldRepoSlug, forkRepoSlug string, defaultBranchOnly bool) {
	body := map[string] any{
		"org_slug":			   orgSlug,
		"slug":                forkRepoSlug,
		"default_branch_only": defaultBranchOnly,
	}
	result, err := Execute1("POST", fmt.Sprintf("repos/%s/%s/fork", orgSlug, oldRepoSlug), body)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func CloneRepo(orgSlug, repoSlug, destDir string) {
	if destDir == "" {
		destDir = repoSlug
	}
	repoResp, err := Execute1("GET", fmt.Sprintf("repos/%s/%s", orgSlug, repoSlug), nil)
	Ensure(err)

	cloneURL := ""
	if v, ok := repoResp["clone_url"].(map[string]any); ok {
		if https, ok := v["https"].(string); ok && https != "" {
			cloneURL = https
		}
		if cloneURL == "" {
			if ssh, ok := v["ssh"].(string); ok {
				cloneURL = ssh
			}
		}
	}
	if cloneURL == "" {
		Ensure(fmt.Errorf("clone_url not found in repo response"))
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		Ensure(err)
	}

	absPath, err := filepath.Abs(destDir)
	Ensure(err)

	cmd := exec.Command("git", "clone", cloneURL, absPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		Ensure(err)
	}
}

func SyncRepo(orgSlug, repoSlug string) {
	repoResp, err := Execute1("GET", "repos/"+orgSlug+"/"+repoSlug, nil)
	Ensure(err)

	parentObj, hasParent := repoResp["parent"].(map[string]any)
	defaultBranch := ""
	if db, ok := repoResp["default_branch"].(string); ok {
		defaultBranch = db
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	if !hasParent {
		Ensure(fmt.Errorf("repository has no parent to sync from"))
	}

	parentSlug := ""
	if ps, ok := parentObj["slug"].(string); ok {
		parentSlug = ps
	}
	parentOrg := ""
	if po, ok := parentObj["organization"].(map[string]any); ok {
		if oslug, ok := po["slug"].(string); ok {
			parentOrg = oslug
		}
	}
	if parentOrg == "" || parentSlug == "" {
		Ensure(fmt.Errorf("cannot determine parent repository coordinates"))
	}

	wd, err := os.Getwd()
	Ensure(err)

	repoDir := filepath.Join(wd, repoSlug)
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		Ensure(fmt.Errorf("local repository directory %s does not exist; run CloneRepo first", repoDir))
	}

	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			Ensure(err)
		}
	}

	// add upstream remote if missing
	remotesCmd := exec.Command("git", "remote")
	remotesCmd.Dir = repoDir
	out, err := remotesCmd.Output()
	Ensure(err)
	remotes := string(out)
	if !containsRemote(remotes, "upstream") {
		parentCloneURL := ""
		if v, ok := parentObj["clone_url"].(map[string]any); ok {
			if https, ok := v["https"].(string); ok && https != "" {
				parentCloneURL = https
			}
			if parentCloneURL == "" {
				if ssh, ok := v["ssh"].(string); ok {
					parentCloneURL = ssh
				}
			}
		}
		if parentCloneURL == "" {
			Ensure(fmt.Errorf("parent clone_url not found"))
		}
		run("git", "remote", "add", "upstream", parentCloneURL)
	}

	run("git", "fetch", "upstream")
	run("git", "checkout", defaultBranch)
	run("git", "merge", "upstream/"+defaultBranch)
	run("git", "push", "origin", defaultBranch)
}

func containsRemote(list string, name string) bool {
	result, _ := filepath.Match("*"+name+"*", list)
	return result && (len(list) > 0 && (StringContains(list, name)))
}

func visSymbol(vis string) string {
    switch strings.ToLower(vis) {
    case "public":
        return "🌐"
    case "internal":
        return "🔒"
    case "private":
        return "🔐"
    default:
        return "·"
    }
}

func prettyTimeShort(ts any) string {
    if ts == nil {
        return ""
    }
    if s, ok := ts.(string); ok && s != "" {
        if t, err := time.Parse(time.RFC3339, s); err == nil {
            return t.Format("2006-01-02")
        }
        return s
    }
    return ""
}

// ListRepoPretty выводит список репозиториев в git-подобном стиле
func ListRepoPretty(orgSlug string) {
    path := fmt.Sprintf("orgs/%s/repos", orgSlug)
    resp, err := Execute1("GET", path, nil)
    Ensure(err)

    var items []any
    if arr, ok := resp["repos"].([]any); ok {
        items = arr
    } else if arr, ok := resp["items"].([]any); ok {
        items = arr
    } else if arr, ok := resp["data"].([]any); ok {
        items = arr
    } else {
        // fallback: print raw
        fmt.Println(ToJson(resp))
        return
    }

    fmt.Printf("Repositories for %s\n\n", orgSlug)
    for _, it := range items {
        m, ok := it.(map[string]any)
        if !ok {
            continue
        }
        id := ShortID(fmtString1(m["id"], m["slug"]))
        name := fmtString1(m["name"], m["slug"])
        vis := fmtString1(m["visibility"])
        lang := fmtString1(m["language"])
        updated := prettyTimeShort(m["last_updated"])
        counters := m["counters"]
        forks, prs, issues := "0", "0", "0"
        if c, ok := counters.(map[string]any); ok {
            if v, ok := c["forks"].(string); ok { forks = v }
            if v, ok := c["pull_requests"].(string); ok { prs = v }
            if v, ok := c["issues"].(string); ok { issues = v }
        }
        desc := fmtString1(m["description"])
        fmt.Printf("%s %s  %-20s  %s  %s\n", id, visSymbol(vis), name, lang, updated)
        fmt.Printf("    ↳ forks:%s  prs:%s  issues:%s\n", forks, prs, issues)
        if desc != "" {
            fmt.Printf("    %s\n", TruncateString(desc, 80))
        }
        fmt.Println()
    }
}

// ViewRepoPretty выводит детали репозитория в читабельной карточке
func ViewRepoPretty(orgSlug, repoSlug string) {
    path := fmt.Sprintf("repos/%s/%s", orgSlug, repoSlug)
    result, err := Execute1("GET", path, nil)
    Ensure(err)

    id := fmtString1(result["id"])
    name := fmtString1(result["name"], result["slug"])
    desc := fmtString1(result["description"])
    visibility := fmtString1(result["visibility"])
    defaultBranch := fmtString1(result["default_branch"])
    lang := fmtString1(result["language"], mapStringField(result, "language", "name"))
    lastUpdated := prettyTimeShort(result["last_updated"])
    isEmpty := fmtString1(result["is_empty"])
    cloneURL := ""
    if cu, ok := result["clone_url"].(map[string]any); ok {
        cloneURL = fmtString1(cu["https"], cu["ssh"])
    }
    parent := ""
    if p, ok := result["parent"].(map[string]any); ok {
        parent = fmtString1(p["organization"], p["slug"], p["id"])
    }

    fmt.Printf("%s\n", name)
    fmt.Printf("repo %s\n\n", id)
    if desc != "" {
        fmt.Println(IndentString(desc, 4))
        fmt.Println()
    }

    fmt.Printf("Visibility: %s\n", visibility)
    fmt.Printf("Language:   %s\n", lang)
    fmt.Printf("Default:    %s\n", defaultBranch)
    fmt.Printf("Updated:    %s\n", lastUpdated)
    fmt.Printf("Empty:      %s\n", isEmpty)
    if parent != "" {
        fmt.Printf("Parent:     %s\n", parent)
    }
    if cloneURL != "" {
        fmt.Printf("Clone:      %s\n", cloneURL)
    }
    fmt.Println()
}

func fmtString1(vals ...any) string {
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
            // ignore
        }
    }
    return ""
}

func mapStringField(m map[string]any, key, subkey string) string {
    if v, ok := m[key]; ok && v != nil {
        if mm, ok := v.(map[string]any); ok {
            if s, ok := mm[subkey].(string); ok {
                return s
            }
        }
    }
    return ""
}