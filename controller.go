package chat_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
)

type Command struct {
	Command string `json:"command"`
	Data map[string](interface{}) `json:"data"`
}

type MessageListReply struct {
	Command string `json:"command"`
	Messages []Message `json:"messages"`
}

type Controller struct {
	rdb *redis.Client
}

func NewController(rdb *redis.Client) *Controller {
	return &Controller{
		rdb,
	}
}

func (c *Controller) HandleIncomingCommand(command Command, client *Client) () {
	ctx := context.Background()
	switch(command.Command) {
	case "chatlog":
		messages, _ := c.rdb.LRange(ctx, "chatlog", -20, -1).Result()
		formattedResponse, _ := json.Marshal(MessageListReply{
			Command:  command.Command,
			Messages: RedisMessagesUnmarshaller(messages),
		})
		client.send <- formattedResponse
	case "sendmessage":
		fmt.Println(command.Data)
		store, _ := json.Marshal(command.Data)
		c.rdb.RPush(ctx, "chatlog", store)
		c.rdb.Publish(ctx, "chatlog_channel", store)
	}

}
