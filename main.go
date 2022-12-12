package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/anaskhan96/soup"
	"google.golang.org/api/youtube/v3"

	"golang.org/x/oauth2/google"
)

type Video struct {
	Book     string
	DVD      string
	Title    string
	Link     string
	FilePath string
}

func main() {

	videos := []Video{}

	links := videoLinks()
	for _, link := range links {
		video := linkToVideo(link.Text(), link.Attrs()["href"])
		videos = append(videos, video)
	}

	downloadVideos(&videos)

	uploadVideo(videos)
}

// downloadVideos downloads the assets from videos[].Link and updated videos[].FilePath
func downloadVideos(videos *[]Video) {
	for _, video := range *videos {
		// Build fileName from fullPath
		fileURL, err := url.Parse(video.Link)
		if err != nil {
			log.Fatal(err)
		}
		path := fileURL.Path
		segments := strings.Split(path, "/")
		fileName := "./downloads/" + segments[len(segments)-1]

		// Create blank file
		file, err := os.Create(fileName)
		if err != nil {
			log.Fatal(err)
		}
		client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}

		fmt.Printf("Downloading %s into %s...", video.Link, fileName)
		// Put content on file
		resp, err := client.Get(video.Link)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		size, err := io.Copy(file, resp.Body)

		defer file.Close()

		fmt.Printf("Downloaded a file %s with size %d", fileName, size)

		video.FilePath = fileName
	}
}

func videoLinks() []soup.Root {

	resp, _ := soup.Get("http://lqtoronto.com/videodlmac.html")

	doc := soup.HTMLParse(resp)

	// All a attribute for DVD1
	videoLinks := doc.Find("div", "id", "dvd1").FindAll("a")

	// Append all attribute for DVD2
	videoLinks = append(videoLinks, doc.Find("div", "id", "dvd2").FindAll("a")...)

	// Append all attribute for DVD3
	videoLinks = append(videoLinks, doc.Find("div", "id", "dvd3").FindAll("a")...)

	return videoLinks

}

func linkToVideo(title string, link string) Video {
	s := strings.Split(link, "_")
	return Video{
		Title: title,
		DVD:   s[len(s)-3],
		Book: map[string]string{
			"BK1": "Book 1",
			"BK2": "Book 2",
			"BK3": "Book 3",
		}[s[len(s)-4]],
		Link: link,
	}
}

func uploadVideo(videos []Video) {
	ctx := context.Background()

	b, err := ioutil.ReadFile("/home/seraf/ytcreds.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/youtube-go-quickstart.json
	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope, youtube.YoutubeUploadScope, youtube.YoutubeScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)
	service, err := youtube.New(client)

	handleError(err, "Error creating YouTube client")

	channel := channelsList(service, []string{"snippet", "contentDetails", "statistics"})

	playlists := playlistsMap(service, []string{"snippet", "contentDetails"})

	syncPlaylists(service, channel.Id, playlists, videos)
}
