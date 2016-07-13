package main

import (
	"pistis/pistis"
	"fmt"
)

func main() {
	pistis.Start("tcp://127.0.0.1:1883")
	pistis.StartHttpServer("http://localhost:8080")
	fmt.Println("running...")
}
