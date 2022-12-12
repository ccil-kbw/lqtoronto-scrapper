package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("youtube-kbw.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

func channelsList(service *youtube.Service, part []string) *youtube.Channel {
	call := service.Channels.List(part)
	call = call.Mine(true)
	response, err := call.Do()
	handleError(err, "")
	fmt.Println(fmt.Sprintf("This channel's ID is %s. Its title is '%s', "+
		"and it has %d views.",
		response.Items[0].Id,
		response.Items[0].Snippet.Title,
		response.Items[0].Statistics.ViewCount))

	return response.Items[0]
}

func playlistsMap(service *youtube.Service, part []string) map[string]*youtube.Playlist {
	call := service.Playlists.List([]string{"snippet", "contentDetails"})
	response, err := call.Mine(true).Do()

	handleError(err, "")

	playlists := map[string]*youtube.Playlist{}
	for _, item := range response.Items {
		playlists[item.Snippet.Title] = item

	}

	return playlists
}

func syncPlaylists(service *youtube.Service, channelID string, playlists map[string]*youtube.Playlist, videos []Video) {

	createIfNotExist := map[string]*youtube.Playlist{
		"Book 1": {
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: "unlisted",
			},
			Snippet: &youtube.PlaylistSnippet{
				ChannelId: channelID,
				Title:     "Book 1",
			},
		},
		"Book 2": {
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: "unlisted",
			},
			Snippet: &youtube.PlaylistSnippet{
				ChannelId: channelID,
				Title:     "Book 2",
			},
		},
		"Book 3": {
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: "unlisted",
			},
			Snippet: &youtube.PlaylistSnippet{
				ChannelId: channelID,
				Title:     "Book 3",
			},
		},
	}

	for k, v := range createIfNotExist {
		if _, ok := playlists[k]; !ok {
			call := service.Playlists.Insert([]string{"status", "snippet"}, v)
			response, err := call.Do()
			handleError(err, "")
			fmt.Println(response.Snippet)
		}
	}

	for _, v := range videos {
		call := service.Videos.Insert([]string{"snippet", "status"}, &youtube.Video{
			Status: &youtube.VideoStatus{
				PrivacyStatus: "unlisted",
			},
			Snippet: &youtube.VideoSnippet{
				ChannelId: channelID,
				Title:     v.Title,
			},
		})

		file, err := os.Open(v.FilePath)
		defer file.Close()
		if err != nil {
			log.Fatalf("Error opening %v: %v", v.FilePath, err)
		}

		response, err := call.Media(file).Do()
		handleError(err, "")

		fmt.Println("Upload successful! Video ID: %v", response.Id)
	}
}
