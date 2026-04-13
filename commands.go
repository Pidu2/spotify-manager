package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/zmb3/spotify/v2"
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

			itemPage, err := client.GetPlaylistItems(ctx, pl.ID, spotify.Limit(100))
			if err != nil {
				return fmt.Errorf("fetch items for playlist %q: %w", pl.Name, err)
			}
			for {
				for _, item := range itemPage.Items {
					if item.IsLocal {
						continue // local files can't be liked
					}
					track := item.Track.Track
					if track == nil {
						continue // skip episodes
					}
					if _, ok := liked[track.ID]; !ok {
						artists := make([]string, len(track.Artists))
						for i, a := range track.Artists {
							artists[i] = a.Name
						}
						fmt.Printf("%s: %s - %s\n", pl.Name, track.Name, strings.Join(artists, ", "))
					}
				}
				if err := client.NextPage(ctx, itemPage); err == spotify.ErrNoMorePages {
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
