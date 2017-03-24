package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
	"treego"
)

func main() {
	api := treego.NewApi("vm.loc:8080", "", treego.ENDPOINT_TCP)
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

	go func() {
		time.Sleep(time.Second * 2)
		data, err := ioutil.ReadFile(os.Args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		event := &treego.Event{Name: "Test Event", From: api.Token, Data: data}

		for {
			//fmt.Println("Writing it")
			api.Emit(event)
			//time.Sleep(time.Millisecond * 100)
		}
	}()

	api.Start(5)
}
