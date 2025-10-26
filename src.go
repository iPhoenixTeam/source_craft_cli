package main

import (
	"fmt"
	"os"

	"phoenix.team/src/cli"
)

var Version = "1.0.0"

func main() {
	cmd := os.Args[1]
	
	switch cmd {
		case "repo":
			cli.DispatchRepo(os.Args[2], os.Args[3:])
		case "issue":
			cli.DispatchIssue(os.Args[2], os.Args[3:])
		case "milestone":
			cli.DispatchMilestone(os.Args[2], os.Args[3:])
		case "workflow":
			cli.DispatchWorkflow(os.Args[2], os.Args[3:])

		default:
		fmt.Println("Unknown command")
		os.Exit(1)
	}

	//test := map[string] any {
	//	"description": "This is a test repository",
	//}

	//execute("PATCH", "repos/id:019a1754-124f-7a87-81c6-7a4393855cf2", test)

	//fmt.Print("hii")
}

