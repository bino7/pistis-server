package main

import (
	"pistis/pistis"
	"fmt"
)


func main() {
	pistis.Start("tcp://192.168.0.137:1883")
	pistis.StartHttpServer("http://localhost:8080")
	fmt.Println("running...")
}
