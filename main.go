package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("minicontainer version 0.1.0")

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: minicontainer run <command> [args...]")
			os.Exit(1)
		}
		containerCommand := os.Args[2]
		containerArgs := os.Args[3:]

		fmt.Printf("Command: %s\n", containerCommand)
		fmt.Printf("Args: %v\n", containerArgs)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: ./minicontainer <command> [options]")
}
