package main

import (
	"fmt"
	"os"
	"strings"

	"phoenix.team/src/cli"
)

var Version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: src <command> [<subcommand> [args...]]")
        os.Exit(2)
    }

    cmd := os.Args[1]

    switch cmd {
    case "--version", "--v", "-v":
        fmt.Println(Version)
        return
    case "--help", "-h", "help":
        fmt.Fprintln(os.Stderr, `Commands:
  repo <cmd> ...
  issue <cmd> ...
  pr <cmd> ...
  milestone <cmd> ...
  workflow <cmd> ...
  report <cmd> ...
  stats <cmd> ...
  auth <cmd> ...
  config <cmd> ...
Use "src <command> --help" for command-specific help.`)

    default:
        if len(os.Args) < 3 {
            fmt.Fprintln(os.Stderr, "missing subcommand")
            os.Exit(2)
        }
        sub := os.Args[2]
        args := os.Args[3:]

        switch cmd {
        case "repo":
            cli.DispatchRepo(sub, args)
        case "issue":
            cli.DispatchIssue(sub, args)
        case "pr":
            cli.DispatchPr(sub, args)
        case "milestone":
            cli.DispatchMilestone(sub, args)
        case "workflow":
            cli.DispatchWorkflow(sub, args)
        case "report":
            cli.DispatchReport(sub, args)
        case "stats":
            cli.DispatchStats(sub, args)
		case "auth":
            cli.DispatchAuth(sub, args)
        case "config":
            cli.DispatchConfig(sub, args)
        default:
            fmt.Fprintln(os.Stderr, "Unknown command:", cmd)
            printCommandHelp(cmd)
			os.Exit(1)
        }
    }
}

func printCommandHelp(cmd string) {
	fmt.Printf("Usage: src %s\n\n", cmd)

	// add brief examples / expanded help for common commands
	switch {
	case strings.HasPrefix(cmd, "repo "):
		fmt.Println("Examples:")
		fmt.Println("  src repo list")
		fmt.Println("  src repo create my-repo")
		fmt.Println("  src repo view org/my-repo")
	case strings.HasPrefix(cmd, "issue "):
		fmt.Println("Examples:")
		fmt.Println("  src issue list")
		fmt.Println("  src issue create org repo \"Issue title\"")
		fmt.Println("  src issue view org repo issue-slug")
	case strings.HasPrefix(cmd, "auth "):
		fmt.Println("Examples:")
		fmt.Println("  src auth login <personal-token>")
		fmt.Println("  src auth logout")
	case strings.HasPrefix(cmd, "config "):
		fmt.Println("Examples:")
		fmt.Println("  src config set editor vim")
		fmt.Println("  src config get editor")
	case strings.HasPrefix(cmd, "code search"):
		fmt.Println("Examples:")
		fmt.Println("  src code search \"TODO\"")
		fmt.Println("  src code search \"func main\"")
	case strings.HasPrefix(cmd, "stats ") || strings.HasPrefix(cmd, "report "):
		fmt.Println("Examples:")
		fmt.Println("  src stats repo org repo")
		fmt.Println("  src report security org repo --top 10")
	default:
		// generic hint
		fmt.Println("Run 'src --help' to see list of available commands.")
	}

	// print flag hint
	fmt.Println("\nFlags:")
	fmt.Println("  --help     Show this help")
	fmt.Println("  --verbose  Enable verbose output")
}