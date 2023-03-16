package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
)

type M map[string]interface{}

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var connections = make(map[string]*WebSocketConnection)

type SocketPayload struct {
	Message string
}

type Message struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
	Age      string
}

func main() {
	e := echo.New()

	e.GET("/", func(ctx echo.Context) error {
		content, err := ioutil.ReadFile("template/chat.html")
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "could not open html")
		}

		return ctx.HTML(http.StatusOK, string(content))
	})
	e.Static("/template", "template")

	e.Any("/ws", func(ctx echo.Context) error {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		currentGorillaConn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
		if err != nil {
			return ctx.String(http.StatusBadRequest, "Could not open websocket connection")
		}

		username := ctx.Request().URL.Query().Get("username")
		age := ctx.Request().URL.Query().Get("age")

		go HandleIO(&WebSocketConnection{
			Conn:     currentGorillaConn,
			Username: username,
			Age:      age,
		})
		return nil
	})

	e.Start(":8080")
}

func HandleIO(currentConn *WebSocketConnection) {
	connections[currentConn.Username] = currentConn
	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)
				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	delete(connections, currentConn.Username)
}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		eachConn.WriteJSON(Message{
			From:    fmt.Sprintf(currentConn.Username + " Age: " + currentConn.Age),
			Type:    kind,
			Message: message,
		})
	}
}
