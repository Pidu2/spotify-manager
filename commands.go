package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	spotify "github.com/zmb3/spotify/v2"
)

func cmdTopTracks(ctx context.Context, client *spotify.Client, rangeArg string) error {
	var timeRange spotify.Range
	switch rangeArg {
	case "short":
		timeRange = spotify.ShortTermRange
	case "medium", "":
		timeRange = spotify.MediumTermRange
	case "long":
		timeRange = spotify.LongTermRange
	default:
		return fmt.Errorf("unknown range %q: use short, medium, or long", rangeArg)
	}

	tracks, err := client.CurrentUsersTopTracks(ctx,
		spotify.Limit(50),
		spotify.Timerange(timeRange),
	)
	if err != nil {
		return fmt.Errorf("fetch top tracks: %w", err)
	}

	fmt.Printf("Top tracks (%s term):\n\n", rangeArg)
	for i, t := range tracks.Tracks {
		artists := make([]string, len(t.Artists))
		for j, a := range t.Artists {
			artists[j] = a.Name
		}
		fmt.Printf("%2d. %s — %s\n", i+1, t.Name, strings.Join(artists, ", "))
	}
	return nil
}

func cmdFollowing(ctx context.Context, client *spotify.Client) error {
	var artists []spotify.FullArtist
	var cursor string
	for {
		page, err := client.CurrentUsersFollowedArtists(ctx, spotify.Limit(50), spotify.After(cursor))
		if err != nil {
			return fmt.Errorf("fetch followed artists: %w", err)
		}
		artists = append(artists, page.Artists...)
		if page.Cursor.After == "" {
			break
		}
		cursor = page.Cursor.After
	}

	fmt.Printf("Following (%d artists):\n\n", len(artists))
	for i, a := range artists {
		fmt.Printf("%2d. %s\n", i+1, a.Name)
	}
	return nil
}

func cmdCleanPlaylist(ctx context.Context, client *spotify.Client) error {
	// Get current user ID to identify personal playlists.
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("fetch current user: %w", err)
	}

	// Build a set of liked track IDs.
	liked := make(map[spotify.ID]struct{})
	savedPage, err := client.CurrentUsersTracks(ctx, spotify.Limit(50))
	if err != nil {
		return fmt.Errorf("fetch liked tracks: %w", err)
	}
	for {
		for _, t := range savedPage.Tracks {
			liked[t.ID] = struct{}{}
		}
		if err := client.NextPage(ctx, savedPage); err == spotify.ErrNoMorePages {
			break
		} else if err != nil {
			return fmt.Errorf("fetch liked tracks: %w", err)
		}
	}

	// Iterate over personal playlists.
	plPage, err := client.CurrentUsersPlaylists(ctx, spotify.Limit(50))
	if err != nil {
		return fmt.Errorf("fetch playlists: %w", err)
	}
	for {
		for _, pl := range plPage.Playlists {
			if pl.Owner.ID != user.ID {
				continue
			}

			trackPage, err := client.GetPlaylistTracks(ctx, pl.ID, spotify.Limit(100))
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping playlist %q: %v\n", pl.Name, err)
				continue
			}
			for {
				for _, item := range trackPage.Tracks {
					if item.IsLocal {
						continue
					}
					if _, ok := liked[item.Track.ID]; !ok {
						artists := make([]string, len(item.Track.Artists))
						for i, a := range item.Track.Artists {
							artists[i] = a.Name
						}
						fmt.Printf("%s: %s - %s\n", pl.Name, item.Track.Name, strings.Join(artists, ", "))
					}
				}
				if err := client.NextPage(ctx, trackPage); err == spotify.ErrNoMorePages {
					break
				} else if err != nil {
					return fmt.Errorf("fetch items for playlist %q: %w", pl.Name, err)
				}
			}
		}
		if err := client.NextPage(ctx, plPage); err == spotify.ErrNoMorePages {
			break
		} else if err != nil {
			return fmt.Errorf("fetch playlists: %w", err)
		}
	}

	return nil
}

func cmdNotFollowing(ctx context.Context, client *spotify.Client, interactive bool) error {
	followed := make(map[spotify.ID]struct{})
	var cursor string
	for {
		page, err := client.CurrentUsersFollowedArtists(ctx, spotify.Limit(50), spotify.After(cursor))
		if err != nil {
			return fmt.Errorf("fetch followed artists: %w", err)
		}
		for _, a := range page.Artists {
			followed[a.ID] = struct{}{}
		}
		if page.Cursor.After == "" {
			break
		}
		cursor = page.Cursor.After
	}

	type artistEntry struct {
		name  string
		count int
	}
	counts := make(map[spotify.ID]*artistEntry)

	savedPage, err := client.CurrentUsersTracks(ctx, spotify.Limit(50))
	if err != nil {
		return fmt.Errorf("fetch liked tracks: %w", err)
	}
	for {
		for _, t := range savedPage.Tracks {
			for _, a := range t.Artists {
				if _, ok := followed[a.ID]; ok {
					continue
				}
				if counts[a.ID] == nil {
					counts[a.ID] = &artistEntry{name: a.Name}
				}
				counts[a.ID].count++
			}
		}
		if err := client.NextPage(ctx, savedPage); err == spotify.ErrNoMorePages {
			break
		} else if err != nil {
			return fmt.Errorf("fetch liked tracks: %w", err)
		}
	}

	type entry struct {
		id    spotify.ID
		name  string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for id, info := range counts {
		entries = append(entries, entry{id: id, name: info.name, count: info.count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].name < entries[j].name
	})

	if !interactive {
		fmt.Printf("Artists with liked tracks you don't follow (%d):\n\n", len(entries))
		for i, e := range entries {
			fmt.Printf("%2d. %-40s %d track(s)\n", i+1, e.name, e.count)
		}
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	newFollows := 0
	for _, e := range entries {
		fmt.Printf("%-40s %d track(s)  Follow? [y/N] ", e.name, e.count)
		if !scanner.Scan() {
			break
		}
		if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			if err := client.FollowArtist(ctx, e.id); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not follow %q: %v\n", e.name, err)
			} else {
				newFollows++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	if newFollows > 0 {
		fmt.Printf("\nNow following %d new artist(s).\n", newFollows)
	}
	return nil
}
