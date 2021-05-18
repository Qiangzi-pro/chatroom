package main

import (
	"context"
	"fmt"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws://localhost:2021/ws", nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close(websocket.StatusInternalError, "internal err")

	err = wsjson.Write(ctx, conn, "Hello WebSocket Server")
	if err != nil {
		panic(err)
	}

	var v interface{}
	err = wsjson.Read(ctx, conn, &v)
	if err != nil {
		panic(err)
	}
	fmt.Printf("receive server message: %v\n", v)

	conn.Close(websocket.StatusNormalClosure, "")
}
