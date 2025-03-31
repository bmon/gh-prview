package main

import (
	"fmt"
	"os"
	"strconv"

	prview "github.com/bmon/gh-prview"
)

func main() {
	// Parse command line arguments for PR number
	var prNumber int
	if len(os.Args) > 1 {
		num, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid PR number: %v\n", err)
			os.Exit(1)
		}
		prNumber = num
	}

	// Call the prview package to handle loading and rendering the PR
	pr, err := prview.LoadPR(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load PR data: %v\n", err)
		os.Exit(1)
	}

	err = prview.RenderPR(os.Stdout, pr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render: %v\n", err)
		os.Exit(1)
	}
}
