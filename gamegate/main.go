package main

import "gogame/base/service"

func main() {
	svr := NewServer()
	service.Run(svr)
}
