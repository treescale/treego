package main

import (
	"treego"
)

func main() {
	treego.NewApi("127.0.0.1:8080", "", treego.ENDPOINT_TCP).Start(1)
}
