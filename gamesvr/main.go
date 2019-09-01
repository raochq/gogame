package main

import "github.com/raochq/gogame/base/service"

func main() {
	svr := &Server{}
	service.Run(svr)
}
