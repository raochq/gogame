package main

import "gogame/base/service"

func main() {
	server := GetServerInstance()
	service.Run(server)
}
