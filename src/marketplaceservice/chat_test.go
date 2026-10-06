package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateChatPersistence(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	buyerID, buyer := f.account(t)
	_, outsider := f.account(t)
	l := f.listing(t, seller)
	base := f.servers[0].URL
	code, b := request(t, base, "POST", "/listings/"+l.ID+"/conversation", buyer, "", nil)
	if code != 200 {
		t.Fatalf("start %d %s", code, b)
	}
	var c map[string]string
	json.Unmarshal(b, &c)
	id := c["id"]
	path := "/conversations/" + id + "/messages"
	code, b = request(t, base, "POST", "/listings/"+l.ID+"/conversation", buyer, "", nil)
	var again map[string]string
	json.Unmarshal(b, &again)
	if code != 200 || again["id"] != id {
		t.Fatal("conversation not deduplicated")
	}
	for _, token := range []string{"", outsider} {
		for _, method := range []string{"GET", "POST"} {
			code, b = request(t, base, method, path, token, "", map[string]string{"body": "intrude", "client_id": "x"})
			if code != 401 && code != 404 {
				t.Fatalf("unauthorized access: %d %s", code, b)
			}
		}
	}
	for i := 0; i < 2; i++ {
		code, b = request(t, base, "POST", path, buyer, "", map[string]string{"body": "Is this available?", "client_id": "retry-1"})
		if code != 200 {
			t.Fatalf("send %d %s", code, b)
		}
	}
	code, b = request(t, base, "POST", path, buyer, "", map[string]string{"body": "Changed", "client_id": "retry-1"})
	if code != 409 {
		t.Fatalf("key conflict: %d", code)
	}
	code, b = request(t, base, "POST", path, seller, "", map[string]string{"body": "Yes, pickup tomorrow.", "client_id": "seller-1"})
	if code != 200 {
		t.Fatalf("reply %d %s", code, b)
	}
	code, b = request(t, f.servers[1].URL, "GET", path, buyer, "", nil)
	var messages []chatMessage
	json.Unmarshal(b, &messages)
	if code != 200 || len(messages) != 2 || messages[0].SenderID != buyerID || messages[1].Body != "Yes, pickup tomorrow." {
		t.Fatalf("persistence/order/dedup %d %s", code, b)
	}
	code, b = request(t, base, "GET", "/conversations", outsider, "", nil)
	if code != 200 || strings.TrimSpace(string(b)) != "[]" {
		t.Fatalf("inbox privacy %d %s", code, b)
	}
	code, b = request(t, base, "POST", path, buyer, "", map[string]string{"body": "  ", "client_id": "blank"})
	if code != 400 {
		t.Fatalf("blank accepted: %d", code)
	}
	code, b = request(t, base, "POST", path, buyer, "", map[string]string{"body": strings.Repeat("x", 2001), "client_id": "long"})
	if code != 400 {
		t.Fatalf("long accepted: %d", code)
	}
	code, b = request(t, base, "POST", "/listings/"+l.ID+"/conversation", seller, "", nil)
	if code != 400 {
		t.Fatalf("self chat accepted %d", code)
	}
}
