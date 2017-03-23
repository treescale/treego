package main

import (
	"fmt"
	"treego"
)

func main() {
	api := treego.NewApi("127.0.0.1:8080", "", treego.ENDPOINT_TCP)
	api.On(treego.EVENT_ON_CONNECTION, func(event *treego.Event) bool {
		fmt.Println("Connected -> ", string(event.Data))
		return true
	})

	api.On(treego.EVENT_ON_CHANNEL_CONNECTION, func(event *treego.Event) bool {
		fmt.Println("New Channel connection openned")
		return true
	})

	api.On(treego.EVENT_ON_CHANNEL_DISCONNECT, func(event *treego.Event) bool {
		fmt.Println("Channel disconnected", string(event.Data))
		return true
	})

	api.On(treego.EVENT_ON_DISCONNECT, func(event *treego.Event) bool {
		fmt.Println("Disconnected ->", string(event.Data))
		return true
	})

	api.Start(2)
}
