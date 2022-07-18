package websockets

import (
	"encoding/json"
	"fmt"

	"github.com/antoniodipinto/ikisocket"
	"github.com/fatih/color"
)

type SocketEvent struct {
	Type      string      `json:"type"`
	Action    string      `json:"action"`
	EventName string      `json:"event_name"`
	Data      interface{} `json:"data"`
}

var SocketInstance *ikisocket.Websocket
var SocketClients = make(map[string]string, 0)

func EmitToSocketId(eventType string, eventName string, action string, data interface{}, uuid string) {

	socketEvent := &SocketEvent{
		Type:      eventType,
		Action:    action,
		EventName: eventName,
		Data:      data,
	}

	event, err := json.Marshal(socketEvent)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(color.CyanString("  Emit to "), color.YellowString(uuid))
	fmt.Println(color.CyanString("  EventName to "), color.YellowString(eventName))
	fmt.Println(color.CyanString("  Event Type: "), color.YellowString(eventType))
	fmt.Println(color.CyanString("  Action: "), color.YellowString(eventType))
	fmt.Println(color.CyanString("  Data: "))
	fmt.Println(data)

	emitSocketErr := SocketInstance.EmitTo(uuid, event)
	if emitSocketErr != nil {
		fmt.Println(emitSocketErr)
	}

}
func EmitTo(socketEvent *SocketEvent, id string) {
	fmt.Println("id: ", SocketClients[id])
	if uuid, found := SocketClients[id]; found {
		fmt.Println("uuid: ", uuid)
		event, err := json.Marshal(socketEvent)
		if err != nil {
			fmt.Println(err)
		}

		emitSocketErr := SocketInstance.EmitTo(uuid, event)
		if emitSocketErr != nil {
			fmt.Println(emitSocketErr)
		}
	}
}
