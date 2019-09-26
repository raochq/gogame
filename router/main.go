package main

import "gogame/base/service"

func main() {
	server := NewServer()
	service.Run(server)
}
