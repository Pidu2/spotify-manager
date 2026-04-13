package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	redirectURL = "http://127.0.0.1:8080/callback"
	tokenFile   = "token.json"
)

var (
	state = "spotify-manager-state"
)

func newAuthenticator() *spotifyauth.Authenticator {
	return spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserTopRead,
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
		),
	)
}

func tokenCachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "spotify-manager")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFile), nil
}

func loadCachedToken() (*oauth2.Token, error) {
	path, err := tokenCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	path, err := tokenCachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func authenticate(ctx context.Context) (*spotify.Client, error) {
	auth := newAuthenticator()

	// Try cached token first
	if token, err := loadCachedToken(); err == nil {
		httpClient := auth.Client(ctx, token)
		client := spotify.New(httpClient)
		// Verify the token still works by fetching the current user
		if _, err := client.CurrentUser(ctx); err == nil {
			return client, nil
		}
	}

	// Run the OAuth2 flow
	tokenCh := make(chan *oauth2.Token, 1)
	errCh := make(chan error, 1)

	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(ctx, state, r)
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			errCh <- err
			return
		}
		fmt.Fprintln(w, "Authentication successful! You can close this tab.")
		tokenCh <- token
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Open this URL in your browser to authenticate:")
	fmt.Println(url)

	var token *oauth2.Token
	select {
	case token = <-tokenCh:
	case err := <-errCh:
		return nil, err
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("warning: shutdown callback server: %v", err)
	}

	if err := saveToken(token); err != nil {
		log.Printf("warning: could not cache token: %v", err)
	}

	client := spotify.New(auth.Client(ctx, token))
	return client, nil
}
