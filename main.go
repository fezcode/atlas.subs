package main

import (
	"fmt"
	"os"

	"github.com/fezcode/atlas.subs/internal/tui"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" {
			fmt.Printf("atlas.subs v%s\n", Version)
			return
		}
		if arg == "-h" || arg == "--help" || arg == "help" {
			showHelp()
			return
		}
	}

	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("crash.txt", []byte(fmt.Sprintf("Panic: %v", r)), 0644)
			os.Exit(1)
		}
	}()

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Atlas Subs - A beautiful terminal-based subtitle searcher and downloader.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  atlas.subs              Start the interactive TUI")
	fmt.Println("  atlas.subs -v           Show version")
	fmt.Println("  atlas.subs -h           Show this help")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  - Search and download subtitles for movies and series")
	fmt.Println("  - Interactive UI powered by Bubble Tea")
	fmt.Println("  - Automatically extracts and saves subtitles")
}
