// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path == "/agent" {
		http.ServeFile(w, r, "agent_home.html")
	}
	if r.URL.Path == "/customer" {
		http.ServeFile(w, r, "customer_home.html")
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()
	http.HandleFunc("/agent", serveHome)
	http.HandleFunc("/customer", serveHome)
	http.HandleFunc("/wsa", func(w http.ResponseWriter, r *http.Request) {
		serveWsa(hub, w, r)
	})
	http.HandleFunc("/wsc", func(w http.ResponseWriter, r *http.Request) {
		serveWsc(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
