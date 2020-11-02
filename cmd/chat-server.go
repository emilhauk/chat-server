package main

import (
	"flag"
	chat_server "github.com/emilhauk/chat-server"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":9001", "http service address")
var redisHost = flag.String("redis", ":6379", "redis service address")

func main() {
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr: *redisHost,
	})

	controller := chat_server.NewController(rdb)

	hub := chat_server.NewHub()
	go hub.Run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		chat_server.ServeWs(hub, w, r, controller)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}






