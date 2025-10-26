package main

import (
    "flag"
    "fmt"
    "os"
)

func main() {
    fs := flag.NewFlagSet("mycmd", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "of %s:\n", os.Args[0])
		fs.PrintDefaults()
	}
    verbose := fs.Bool("v", false, "enable verbose")
    name := fs.String("name", "world", "name to greet")
    count := fs.Int("n", 1, "times to greet")

    // парсим только аргументы после имени команды, например os.Args[1:]
    if err := fs.Parse(os.Args[1:]); err != nil {
        fmt.Fprintln(os.Stderr, "parse error:", err)
        os.Exit(2)
    }
	fmt.Print(fs.Args())
    for i := 0; i < *count; i++ {
        if *verbose {
            fmt.Printf("hello (%d/%d) %s\n", i+1, *count, *name)
        } else {
            fmt.Printf("hello %s\n", *name)
        }
    }
}