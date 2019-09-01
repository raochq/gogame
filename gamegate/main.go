package main

import "github.com/raochq/gogame/base/service"

func main() {
	svr := NewServer()
	service.Run(svr)
}
