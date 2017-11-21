// https://schier.co/blog/2015/04/26/a-simple-web-scraper-in-go.html
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"golang.org/x/net/html"
)

func saveImage(imgURL string, saveResult chan string) {
	resp, err := http.Get(imgURL)
	if err != nil {
		saveResult <- fmt.Sprintf("Get image from url %s error: %e", imgURL, err)
	}
	defer resp.Body.Close()

	imgName := path.Base(imgURL)

	file, err := os.Create(imgName)
	if err != nil {
		saveResult <- fmt.Sprintf("Create image %s error: %e", imgName, err)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		saveResult <- fmt.Sprintf("Save image %s error: %e", imgName, err)
	}

	file.Close()
	saveResult <- fmt.Sprintf("Save image %s complete", imgName)

}

func getLinkURL(t html.Token) (bool, string) {
	var href string
	var ok bool
	for _, url := range t.Attr {
		if url.Key == "src" && strings.Index(url.Val, "http") != -1 {
			href = url.Val
			ok = true
		}
	}

	return ok, href
}

func getAllImgURL(url string, imgChURL chan string, chFinished chan bool) {
	resp, err := http.Get(url)
	defer func() {
		chFinished <- true
	}()

	if err != nil {
		fmt.Printf("Error: Failed to crawl %s \n", url)
		return
	}

	b := resp.Body
	defer b.Close()

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return

		case tt == html.StartTagToken:
			t := z.Token()

			imgLink := t.Data == "img"
			if !imgLink {
				continue
			}

			ok, imgURL := getLinkURL(t)
			if !ok {
				continue
			}

			imgChURL <- imgURL
		}
	}

}

func main() {

	foundURLs := make(map[string]bool)

	chURLS := make(chan string, 1)
	chFinished := make(chan bool)
	urls := os.Args[1:]

	for _, url := range urls {
		go getAllImgURL(url, chURLS, chFinished)
	}

	for c := 0; c < len(urls); {
		select {
		case url := <-chURLS:
			foundURLs[url] = true
		case <-chFinished:
			c++
		}
	}

	saveImgRes := make(chan string, 1)
	for url := range foundURLs {
		go saveImage(url, saveImgRes)
	}

	for url := range foundURLs {
		fmt.Printf("URL: %s -> %s\n", url, <-saveImgRes)
	}

	close(chURLS)

}
