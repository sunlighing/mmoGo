package states

import (
	"fmt"
	"log"
	"server/internal/server"
	"server/pkg/packets"
)

type Connected struct {
	client server.ClientInterfacer

	logger *log.Logger
}

func (c *Connected) Name() string {
	return "Connected"
}

func (c *Connected) SetClient(client server.ClientInterfacer) {
	c.client = client
	loggingPrefix := fmt.Sprintf("Client %d [%s]:", client.Id(), c.Name())

	c.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
}

func (c *Connected) OnEnter() {
	c.client.SocketSend(packets.NewId(c.client.Id()))
}

func (c *Connected) HandlerMessage(senderId uint64, message packets.Msg) {
	if senderId == c.client.Id() {
		c.client.Broadcast(message)
	} else {
		//如果不是就发给自己
		c.client.SocketSendAs(message, senderId)
	}
}

func (c *Connected) OnExit() {

}
