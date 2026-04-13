# spotify-manager

A personal CLI tool for managing and inspecting your Spotify library.

> **Disclaimer:** This project has been 100% written by [Claude Code](https://claude.ai/code).

---

## Setup

### 1. Create a Spotify app

Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) and create an app. Under its settings, add the following redirect URI:

```
http://127.0.0.1:8080/callback
```

### 2. Set environment variables

```bash
export SPOTIFY_ID=your_client_id
export SPOTIFY_SECRET=your_client_secret
```

### 3. Build

```bash
go mod tidy
go build -o spotify-manager .
```

### 4. Authenticate

The first time you run any command, a browser URL will be printed. Open it, approve access, and the token will be cached at `~/.config/spotify-manager/token.json`. Subsequent runs skip this step.

---

## Commands

### `top-tracks`

Prints your top 50 tracks for a given time range.

```
spotify-manager top-tracks [short|medium|long]
```

| Range    | Period          |
|----------|-----------------|
| `short`  | Last ~4 weeks   |
| `medium` | Last ~6 months (default) |
| `long`   | All time        |

**Examples:**

```bash
spotify-manager top-tracks short
spotify-manager top-tracks long
```

---

### `clean-playlist`

Finds tracks in your personal playlists that you have not liked. Useful for auditing playlists and keeping your library consistent.

Outputs one line per offending track in the format:

```
<playlist name>: <track name> - <artist name>
```

Local files and podcast episodes are ignored.

**Example:**

```bash
spotify-manager clean-playlist
```

```
Chill Mix: Some Song - Some Artist
Road Trip: Another Track - Another Artist
```
