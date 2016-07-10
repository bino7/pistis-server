package main

import (
	"pistis/pistis"
)

func main() {
	pistis.Start("tcp://127.0.0.1:1883")
	pistis.StartHttpServer("http://localhost:8080")
	/*done:=make(chan bool)
	go func(){
		d,_:=time.ParseDuration("1s")
		fmt.Println("server running")
		for range time.Tick(d){
			if s.IsRunning(){
				done <- true
			}else{
				fmt.Print(".")
			}
		}
	}()*/
	/*<-done*/
}
