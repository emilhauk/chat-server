package chat_server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func checkOrigin(r *http.Request) (bool) {
	return true
}

// serveWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, c *Controller) {
	upgrader.CheckOrigin = checkOrigin
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Connected to", r.RemoteAddr, r.UserAgent())
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), controller: *c}
	client.hub.register <- client
	client.conn.SetCloseHandler(client.closeHandler)
	client.isAlive = true

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
	go client.subscriptionPump()
}
