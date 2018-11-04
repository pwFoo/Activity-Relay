package main

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	faktory "github.com/contribsys/faktory/client"
	"github.com/go-redis/redis"
	"github.com/yukimochi/Activity-Relay/ActivityPub"
	"github.com/yukimochi/Activity-Relay/KeyLoader"
)

var Hostname *url.URL
var Hostkey *rsa.PrivateKey
var FakClient *faktory.Client
var RedClient *redis.Client

// Actor : Relay's Actor
var Actor activitypub.Actor

// WebfingerResource : Relay's Webfinger resource
var WebfingerResource activitypub.WebfingerResource

func main() {
	pemPath := os.Getenv("ACTOR_PEM")
	if pemPath == "" {
		panic("Require ACTOR_PEM environment variable.")
	}
	relayDomain := os.Getenv("RELAY_DOMAIN")
	if relayDomain == "" {
		panic("Require RELAY_DOMAIN environment variable.")
	}
	relayBind := os.Getenv("RELAY_BIND")
	if relayBind == "" {
		relayBind = "0.0.0.0:8080"
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "127.0.0.1:6379"
	}

	var err error
	Hostkey, err = keyloader.ReadPrivateKeyRSAfromPath(pemPath)
	if err != nil {
		panic("Can't read Hostkey Pemfile")
	}
	Hostname, err = url.Parse("https://" + relayDomain)
	if err != nil {
		panic("Can't parse Relay Domain")
	}
	FakClient, err = faktory.Open()
	if err != nil {
		panic("Can't connect Faktory server")
	}
	RedClient = redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	Actor = activitypub.GenerateActor(Hostname, &Hostkey.PublicKey)
	WebfingerResource = activitypub.GenerateWebfingerResource(Hostname, &Actor)

	http.HandleFunc("/.well-known/webfinger", handleWebfinger)
	http.HandleFunc("/actor", handleActor)
	http.HandleFunc("/inbox", handleInbox)

	fmt.Println("Open services" + relayBind)
	log.Fatal(http.ListenAndServe(relayBind, nil))
}
