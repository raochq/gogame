package main

import "gogame/base/service"

func main() {
	svr := &Server{}
	service.Run(svr)
}
