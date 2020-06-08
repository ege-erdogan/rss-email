package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

const days = 7
const feedsURL = "https://raw.githubusercontent.com/ege-erdogan/rss-email/master/feeds.txt"

func main() {
	// lambda.Start(HandleRequest)
	HandleRequest()
}

// HandleRequest called to handle AWS lambda request
func HandleRequest() {
	var blocks []string
	var message string

	dateThreshold := time.Now().AddDate(0, 0, -days)

	resp, err := http.Get(feedsURL)
	check(err)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	check(err)

	urls := strings.Split(string(data), "\n")

	feedBlocks := make(chan string)

	for _, url := range urls {
		if len(url) > 5 { //TODO: better validity check
			go func(url string) {
				fetch(url, dateThreshold, feedBlocks)
			}(url)
		}
	}

	for i := 0; i < len(urls); i++ { //TODO: better error handling than in fetch
		feedBlock := <-feedBlocks
		blocks = append(blocks, feedBlock)
		message = GenerateMessage(blocks)
	}

	fmt.Println(message)
	// send(os.Getenv("RSS_TARGET"), msg)
}

func fetch(url string, threshold time.Time, out chan string) {
	data, err := gofeed.NewParser().ParseURL(url)
	if err != nil {
		out <- "An error occured."
	}

	feed := Feed{Title: data.Title, Link: data.Link}

	for i := 0; i < len(data.Items); i++ {
		if data.Items[i].PublishedParsed.After(threshold) {
			post := Post{Title: data.Items[i].Title,
				Link:       data.Items[i].Link,
				DateString: data.Items[i].PublishedParsed.Format("Jan 2")}
			feed.Posts = append(feed.Posts, post)
		}
	}

	out <- GenerateHTMLFeedBlock(feed)
}

func send(to, body string) {
	username := os.Getenv("EMAIL_NAME")
	password := os.Getenv("EMAIL_PASS")
	from := os.Getenv("EMAIL_FROM")

	msg := "From: " + from + "\n"
	msg += "To: " + to + "\n"
	msg += "Content-Type: text/html\n"
	msg += "Subject: RSS FEEDS\n\n"
	msg += body

	// email-smtp.us-east-1.amazonaws.com
	err := smtp.SendMail("email-smtp.us-east-1.amazonaws.com:587",
		smtp.PlainAuth("", username, password, "email-smtp.us-east-1.amazonaws.com"),
		from, []string{to}, []byte(msg))
	check(err)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
