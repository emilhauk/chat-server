package chat_server

import "encoding/json"

type Message struct {
	Text string `json:"text"`
}

func RedisMessagesUnmarshaller(messages []string) []Message {
	var unmarshalledMessages []Message
	for _, message := range messages {
		var unmarshalledMessage Message;
		err := json.Unmarshal([]byte(message), &unmarshalledMessage)
		if(err == nil) {
			unmarshalledMessages = append(unmarshalledMessages, unmarshalledMessage)
		}
	}
	return unmarshalledMessages
}
