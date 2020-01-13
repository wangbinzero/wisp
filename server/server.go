package server

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"wisp/common"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true,
}

func Echo(res http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		return
	}
	common.ChanMap["abc"] = c
	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println(message)

	}
}
