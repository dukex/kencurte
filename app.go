package main

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"menteslibres.net/gosexy/redis"
)

var api *anaconda.TwitterApi
var client *redis.Client

func main() {
	client = redis.New()

	portInt, _ := strconv.Atoi(os.Getenv("REDIS_PORT"))

	err := client.Connect(os.Getenv("REDIS_SERVER"), uint(portInt))
	if err != nil {
		log.Fatal(err)
	}

	anaconda.SetConsumerKey(os.Getenv("CONSUMER_KEY"))
	anaconda.SetConsumerSecret(os.Getenv("CONSUMER_SECRET"))

	api = anaconda.NewTwitterApi(os.Getenv("USER_KEY"), os.Getenv("USER_SECRET"))

	tweetC := make(chan anaconda.Tweet)
	tweetTickerC := time.NewTicker(2 * time.Hour).C

	for {
		select {
		case tweet := <-tweetC:
			api.Retweet(tweet.Id, false)
		case <-tweetTickerC:
			go request(tweetC)
		}
	}
}

func request(tweetC chan<- anaconda.Tweet) {
	v := url.Values{}
	v.Set("count", "10")
	v.Set("since_id", getSinceID())

	searchResult, err := api.GetSearch("\"quem curte\" -RT ?", v)
	if err != nil {
		return
	}
	for _, tweet := range searchResult {
		tweetC <- tweet
	}
	if len(searchResult) > 0 {
		client.Set("last_tweet_id", searchResult[0].Id)
	}
}

func getSinceID() string {
	sinceID, err := client.Get("last_tweet_id")
	if err != nil {
		return "0"
	}
	return sinceID
}
