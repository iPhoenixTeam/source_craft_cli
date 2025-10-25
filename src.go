package main

import (
	"fmt"
	"os"

	"phoenix.team/src/cli"
)

func main() {
	cmd := os.Args[1]
	
	switch cmd {
		case "repo":
			subcmd := os.Args[2]

			switch subcmd {
			case "list":
				cli.ListRepo(os.Args[3])
			case "create":
				cli.CreateRepo(os.Args[3], os.Args[4], os.Args[4], "", cli.Public, false)
			case "fork":
				cli.ForkRepo(os.Args[3], os.Args[4], os.Args[5], true)
			case "view":
				cli.ViewRepo(os.Args[3], os.Args[4])
		}	
		case "issue":
			
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

