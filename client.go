package chat_server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	isAlive bool

	// Controller to handle incoming commands
	controller Controller
}

func (c *Client) closeHandler(code int, text string) error {
	fmt.Printf("Connection closed (%d): %s\n", code, text)
	c.isAlive = false
	c.hub.unregister <- c
	c.conn.Close()
	return nil
}

func (c *Client) subscriptionPump() {
	defer func() {
		fmt.Println("Closing subscriptionPump")
	}()
	ctx := context.Background()
	subscription :=c.controller.rdb.Subscribe(ctx, "chatlog_channel")
	for {
		select {
		case msg, ok := <-subscription.Channel():
			if !ok {
				fmt.Printf("Nogood %v", msg)
				break
			}

			if !c.isAlive {
				return;
			}

			fmt.Printf("%s: message: %s\n", msg.Channel, msg.Payload)
			formattedResponse, _ := json.Marshal(MessageListReply{
				Command:  "receivedmessage",
				Messages: RedisMessagesUnmarshaller([]string{msg.Payload}),
			})

			c.send <- formattedResponse
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {fmt.Println("Closing readPump")}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if !c.isAlive {
			return
		}

		data = bytes.TrimSpace(bytes.Replace(data, newline, space, -1))
		var command Command
		err = json.Unmarshal(data, &command)
		if err != nil {
			fmt.Errorf("error: %v", err)
			return
		}
		c.controller.HandleIncomingCommand(command, c)

		//c.hub.broadcast <- data
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		fmt.Println("Closing writePump")
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !c.isAlive {
				fmt.Printf("Channel closed")
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

