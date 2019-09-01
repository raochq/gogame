package main

import "github.com/raochq/gogame/base/service"

func main() {
	server := NewServer()
	service.Run(server)
}
