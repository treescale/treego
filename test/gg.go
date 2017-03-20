package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	data := make([]byte, 30)
	binary.BigEndian.PutUint32(data[6:], uint32(400))
	fmt.Println(data)
}
