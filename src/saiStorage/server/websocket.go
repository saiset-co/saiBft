package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan []byte)
var upgrader = websocket.Upgrader{}

func (s Server) WSProcess() {
	for {
		msg := <-broadcast
		s.handleWebSocketRequest(msg)

		for k, client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			time.Sleep(3 * time.Millisecond)

			if err != nil {
				_ = client.Close()
				delete(clients, k)
			}
		}
	}
}

func (s Server) handleWSConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	wss, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
	}

	defer func(wss *websocket.Conn) {
		cErr := wss.Close()

		if cErr != nil {
			fmt.Println(cErr)
		}
	}(wss)

	clients[wss.RemoteAddr().String()] = wss

	for {
		_, msg, rErr := wss.ReadMessage()

		if rErr != nil {
			return
		}

		broadcast <- msg
	}
}
