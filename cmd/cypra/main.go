// Package main provides the Cypra CLI entrypoint.
package main

import (
	"fmt"
	"os"

	_ "github.com/watzon/cypra/dashboard"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("cypra %s (%s)\n", version, commit)
		return
	}

	fmt.Fprintln(os.Stderr, "cypra: command not implemented yet")
	os.Exit(2)
}
