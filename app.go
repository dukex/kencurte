package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/op/go-logging"
	"gopkg.in/redis.v2"
)

var api *anaconda.TwitterApi
var client *redis.Client
var logg = logging.MustGetLogger("ken")

var format = "%{color}%{time:15:04:05.000000} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"

func main() {
	client = redis.NewTCPClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	syslogBackend, err := logging.NewSyslogBackend("")
	if err != nil {
		logg.Fatal(err)
	}
	logging.SetBackend(logBackend, syslogBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.DEBUG, "ken")
	logg.Info("Starting")

	anaconda.SetConsumerKey(os.Getenv("CONSUMER_KEY"))
	anaconda.SetConsumerSecret(os.Getenv("CONSUMER_SECRET"))

	api = anaconda.NewTwitterApi(os.Getenv("USER_KEY"), os.Getenv("USER_SECRET"))

	tweetC := make(chan anaconda.Tweet)
	tweetTickerC := time.NewTicker(2 * time.Hour).C
	go request(tweetC)

	go func() {
		for {
			select {
			case tweet := <-tweetC:
				logg.Info("Retwetted " + tweet.IdStr)
				t, err := api.Retweet(tweet.Id, false)
				if err != nil {
					logg.Error(err.Error())
				} else {
					log.Println(t)
				}
			case <-tweetTickerC:
				logg.Info("Ticker")
				go request(tweetC)
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Kencurte?", "")
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func request(tweetC chan<- anaconda.Tweet) {
	v := url.Values{}
	v.Set("count", "10")
	v.Set("since_id", getSinceID())

	logg.Info("Search with " + v.Encode())
	searchResult, err := api.GetSearch("\"quem curte\" -RT ?", v)
	if err != nil {
		return
	}

	logg.Info("Found " + string(len(searchResult)) + " Tweets")
	for _, tweet := range searchResult {
		tweetC <- tweet
	}
	if len(searchResult) > 0 {
		client.Set("last_tweet_id", searchResult[0].IdStr)
	}
}

func getSinceID() string {
	result := client.Get("last_tweet_id")
	if result.Err() != nil {
		return "0"
	}
	return result.String()
}
