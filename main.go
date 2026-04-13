package main

import (
	"context"
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: spotify-manager <command> [args]

Commands:
  top-tracks [short|medium|long]   Your top tracks (default: medium)
  clean-playlist                   Tracks in your playlists that are not liked

`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := authenticate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth: %v\n", err)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "top-tracks":
		rangeArg := "medium"
		if len(args) > 0 {
			rangeArg = args[0]
		}
		if err := cmdTopTracks(ctx, client, rangeArg); err != nil {
			fmt.Fprintf(os.Stderr, "top-tracks: %v\n", err)
			os.Exit(1)
		}
	case "clean-playlist":
		if err := cmdCleanPlaylist(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "clean-playlist: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", cmd)
		usage()
		os.Exit(1)
	}
}
