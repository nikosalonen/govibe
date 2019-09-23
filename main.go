package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tkanos/gonfig"
	"gopkg.in/irc.v3"
	"mvdan.cc/xurls"
)

//Configuration for irc connection
type Configuration struct {
	URL     string
	Port    string
	Nick    string
	User    string
	Name    string
	Channel string
}

func main() {

	configuration := Configuration{}
	confErr := gonfig.GetConf("config.json", &configuration)
	if confErr != nil {
		fmt.Println(confErr)
		os.Exit(500)
	}
	ircURL := configuration.URL + ":" + configuration.Port
	conn, err := net.Dial("tcp", ircURL)
	if err != nil {
		log.Fatalln(err)
	}
	rx := xurls.Relaxed()
	re := regexp.MustCompile(`http://|https://`)
	re2 := regexp.MustCompile(`://`)

	config := irc.ClientConfig{
		Nick: configuration.Nick,
		User: configuration.User,
		Name: configuration.Name,
		Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
			if m.Command == "001" {
				// 001 is a welcome event, so we join channels there
				c.Write("JOIN " + configuration.Channel)
			} else if m.Command == "PRIVMSG" && c.FromChannel(m) {
				// Create a handler on all messages.
				url := rx.FindString(m.Trailing())
				if len(url) > 0 {
					client := &http.Client{
						Timeout: 30 * time.Second,
					}
					isHTTP := re.MatchString(url)
					isSomethigElse := re2.MatchString(url)
					if !isHTTP && isSomethigElse {
						return
					} else if !isHTTP && !isSomethigElse {
						url = "http://" + url
					}

					resp, err := client.Get(url)
					if err != nil {
						return
					}
					defer resp.Body.Close()

					document, err := goquery.NewDocumentFromReader(resp.Body)

					title := document.Find("title").Text()

					if len(title) > 0 {

						c.WriteMessage(&irc.Message{
							Command: "PRIVMSG",
							Params: []string{
								m.Params[0],
								strings.TrimSpace(title),
							},
						})
					}
				}
			}
		}),
	}

	// Create the client
	client := irc.NewClient(conn, config)
	err = client.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
