package main

import "gogame/base/service"

func main() {
	svr := GetGateServerInstance()
	service.Run(svr)
}
