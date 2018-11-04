package main

import (
	"crypto/rsa"
	"fmt"
	"net/url"
	"os"
	"unsafe"

	worker "github.com/contribsys/faktory_worker_go"
	"github.com/yukimochi/Activity-Relay/ActivityPub"
	"github.com/yukimochi/Activity-Relay/KeyLoader"
)

// Hostname : Hostname of Relay
var Hostname *url.URL

// Hostkey : PrivateKey of Relay
var Hostkey *rsa.PrivateKey

// Actor : Relay's Actor
var Actor activitypub.Actor

func relayActivity(ctx worker.Context, args ...interface{}) error {
	fmt.Println("Working on relay job", ctx.Jid())
	inbox := args[0].(string)
	data := args[1].(string)
	_ = activitypub.MirrorActivity(inbox, Actor.ID, *(*[]byte)(unsafe.Pointer(&data)), Hostkey)
	return nil
}

func registorActivity(ctx worker.Context, args ...interface{}) error {
	fmt.Println("Working on registor job", ctx.Jid())
	inbox := args[0].(string)
	data := args[1].(string)
	_ = activitypub.SendActivity(inbox, Actor.ID, *(*[]byte)(unsafe.Pointer(&data)), Hostkey)
	return nil
}

func main() {
	pemPath := os.Getenv("ACTOR_PEM")
	if pemPath == "" {
		panic("Require ACTOR_PEM environment variable.")
	}
	relayDomain := os.Getenv("RELAY_DOMAIN")
	if relayDomain == "" {
		panic("Require RELAY_DOMAIN environment variable.")
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
	Actor = activitypub.GenerateActor(Hostname, &Hostkey.PublicKey)

	relayManager := worker.NewManager()
	relayManager.Register("RelayActivity", relayActivity)
	relayManager.Register("RegistorActivity", registorActivity)
	relayManager.Concurrency = 20
	relayManager.Queues = []string{"relay", "registor"}

	// Start processing jobs, this method does not return
	relayManager.Run()
}
