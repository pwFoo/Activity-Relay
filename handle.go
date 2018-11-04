package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	faktory "github.com/contribsys/faktory/client"
	"github.com/yukimochi/Activity-Relay/ActivityPub"
)

func handleWebfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query()["resource"]
	if r.Method == "GET" && len(resource) == 0 {
		w.WriteHeader(400)
		w.Write(nil)
	} else {
		request := resource[0]
		if request == WebfingerResource.Subject {
			wfresource, err := json.Marshal(&WebfingerResource)
			if err != nil {
				panic(err)
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(wfresource)
		} else {
			w.WriteHeader(404)
			w.Write(nil)
		}
	}
}

func handleActor(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		actor, err := json.Marshal(&Actor)
		if err != nil {
			panic(err)
		}
		w.Header().Add("Content-Type", "application/activity+json")
		w.WriteHeader(200)
		w.Write(actor)
	} else {
		w.WriteHeader(400)
		w.Write(nil)
	}
}

func contains(entries interface{}, finder string) bool {
	switch entry := entries.(type) {
	case string:
		return entry == finder
	case []string:
		for i := 0; i < len(entry); i++ {
			if entry[i] == finder {
				return true
			}
		}
		return false
	}
	return false
}

func pushRelayJob(sourceInbox string, refBytes []byte) {
	receivers, _ := RedClient.Keys("subscription:*").Result()
	for _, receiver := range receivers {
		if sourceInbox != strings.Replace(receiver, "subscription:", "", 1) {
			inboxURL, _ := RedClient.HGet(receiver, "inbox_url").Result()
			job := faktory.NewJob("RelayActivity", inboxURL, string(refBytes))
			job.Queue = "relay"
			job.Priority = 5
			_ = FakClient.Push(job)
		}
	}
}

func pushRegistorJob(inboxURL string, refBytes []byte) {
	job := faktory.NewJob("RegistorActivity", inboxURL, string(refBytes))
	job.Queue = "registor"
	job.Priority = 5
	_ = FakClient.Push(job)
}

func handleInbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		activity, actor, body, err := decodeActivity(r)
		if err != nil {
			w.WriteHeader(400)
			w.Write(nil)
		} else {
			switch activity.Type {
			case "Follow":
				if contains(activity.Object, "https://www.w3.org/ns/activitystreams#Public") {
					domain, _ := url.Parse(activity.Actor)
					resp := activitypub.GenerateActivityResponse(Hostname, domain, "Accept", *activity)
					//resp := activitypub.GenerateActivityResponse(Hostname, domain, "Reject", *activity)

					fmt.Println("Follow Request by", activity.Actor)
					RedClient.HSet("subscription:"+domain.Host, "inbox_url", actor.Endpoints.SharedInbox)
					jsonData, _ := json.Marshal(&resp)
					pushRegistorJob(actor.Inbox, jsonData)
				} else {
					w.WriteHeader(400)
					w.Write([]byte("Follow only allowed for https://www.w3.org/ns/activitystreams#Public"))
				}
			case "Undo":
				nestedActivity, _ := activitypub.DescribeNestedActivity(activity.Object)
				if nestedActivity.Type == "Follow" && nestedActivity.Actor == activity.Actor {
					domain, _ := url.Parse(activity.Actor)
					fmt.Println("Unfollow Request by", activity.Actor)
					RedClient.Del("subscription:" + domain.Host)
					w.WriteHeader(202)
					w.Write(nil)
				} else {
					domain, _ := url.Parse(activity.Actor)
					pushRelayJob(domain.Host, body)
					fmt.Println("Relay status from", activity.Actor)
					w.WriteHeader(202)
					w.Write(nil)
				}
			case "Create", "Update", "Delete", "Announce":
				domain, _ := url.Parse(activity.Actor)
				pushRelayJob(domain.Host, body)
				fmt.Println("Relay status from", activity.Actor)
				w.WriteHeader(202)
				w.Write(nil)
			}
		}
	default:
		w.WriteHeader(400)
		w.Write(nil)
	}
}
