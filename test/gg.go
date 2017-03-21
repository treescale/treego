package main

import (
	"treego"
)

func main() {
	treego.NewApi("api1", "127.0.0.1:8080").Start()
}
