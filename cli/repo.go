package cli

import (
	"flag"
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

func printRepoHelp() {
    fmt.Fprintln(os.Stderr, `repo commands:
  repo list [org]
  repo create <org> <name> [--desc DESC] [--visibility public|private|internal]
  repo fork <srcOrg> <srcRepo> <newSlug> [--default-branch-only]
  repo view <org> <repo> [--json] [--v]
Use "repo <command> --help" for command-specific flags.`)
}

func DispatchRepo(command string, args []string) {
	switch command {
	case "list":
        fs := NewCmd("repo list", "Usage: %s list [org]\n", flag.ContinueOnError)
        
        if err := fs.Parse(args); err == nil {
			ListRepo(fs.Arg(0))
        }

    case "create":
        fs := NewCmd("repo create", "Usage: %s create <org> <name> [options]\n", flag.ContinueOnError)

        desc := fs.String("desc", "", "repository description")
		createReadme := fs.Bool("create-readme", false, "create readme file")
        visibility := fs.String("visibility", string(RepoPublic), "visibility: public|private|internal")
        
		if err := fs.Parse(args); err == nil {
			rem := Require(fs, 2, "Usage: %s create <org> <name> [options]")
        
			var vis RepoVisibility
			switch *visibility {
				case string(RepoPrivate):
					vis = RepoPrivate
				case string(RepoInternal):
					vis = RepoInternal
				default:
					vis = RepoPublic
			}

			CreateRepo(rem[0], rem[1], *desc, vis, *createReadme)
		}

    case "fork":
        fs := NewCmd("repo fork", "Usage: %s fork <srcOrg> <srcRepo> <newSlug> [options]\n", flag.ContinueOnError)
        
        defaultOnly := fs.Bool("default-branch-only", true, "fork")
        
		if err := fs.Parse(args); err == nil {
            rem := Require(fs, 3, "usage: fork requires <srcOrg> <srcRepo> <newSlug>")
			
			ForkRepo(rem[0], rem[1], rem[2], *defaultOnly)
        }
        
    case "view":
        fs := NewCmd("repo view", "Usage: %s view <org> <repo> [options]\n", flag.ContinueOnError)
        
		if err := fs.Parse(args); err == nil {
            rem := Require(fs, 2, "usage: view requires <org> <repo>")

			ViewRepo(rem[0], rem[1])
        }

    case "--help", "-h", "help", "":
        printRepoHelp()

    default:
        printRepoHelp()
	}
}


func ListRepo(orgSlug string) {
    path := fmt.Sprintf("/orgs/%s/repos", orgSlug)
    resp, err := DoRequest("GET", path, nil)
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

func CreateRepo(orgSlug, repoSlug, description string, visibility RepoVisibility, createReadme bool) {
	body := map[string] any {
		"name":        repoSlug,
		"slug":        repoSlug,
		"description": description,
		"visibility":  visibility,
		"init_settings": map[string]any{
			"default_branch": "",
			"create_readme":  createReadme,
		},
	}
	result, err := DoRequest("POST", fmt.Sprintf("/orgs/%s/repos", orgSlug), body)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func ForkRepo(orgSlug, oldRepoSlug, forkRepoSlug string, defaultBranchOnly bool) {
	body := map[string] any{
		"org_slug":			   orgSlug,
		"slug":                forkRepoSlug,
		"default_branch_only": defaultBranchOnly,
	}
	result, err := DoRequest("POST", fmt.Sprintf("/repos/%s/%s/fork", orgSlug, oldRepoSlug), body)
	Ensure(err)
	fmt.Println(ToJson(result))
}

func CloneRepo(orgSlug, repoSlug, destDir string) {
	if destDir == "" {
		destDir = repoSlug
	}
	repoResp, err := DoRequest("GET", fmt.Sprintf("/repos/%s/%s", orgSlug, repoSlug), nil)
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
	repoResp, err := DoRequest("GET", "/repos/"+orgSlug+"/"+repoSlug, nil)
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
    case "internal":
        return "🔒 internal"
    case "private":
        return "🔐 private"
    default:
        return "🌐 public"
    }
}

func ViewRepo(orgSlug, repoSlug string) {
    path := fmt.Sprintf("/repos/%s/%s", orgSlug, repoSlug)
    result, err := DoRequest("GET", path, nil)
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