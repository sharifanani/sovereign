package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sovereign-cli <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  setup    Run the interactive setup wizard")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		// TODO: Run setup wizard
		fmt.Println("Sovereign setup wizard")
		fmt.Println("This will guide you through setting up your Sovereign server.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
