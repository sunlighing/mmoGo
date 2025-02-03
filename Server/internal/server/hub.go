package server

import (
	"log"
	"net/http"
	"server/internal/server/objects"
	"server/pkg/packets"
)

type ClientInterfacer interface {
	Initialize(id uint64)
	Id() uint64

	//处理消息
	ProcessMessage(senderId uint64, msg packets.Msg)

	//发送消息
	SocketSend(msg packets.Msg)

	SocketSendAs(message packets.Msg, senderId uint64)

	// 前端消息通过别人的处理
	PassToPeer(message packets.Msg, peerId uint64)

	//持续化WebSocket
	ReadPump()
	WritePump()

	Close(reson string)

	Broadcast(message packets.Msg)
}

type Hub struct {
	Clients *objects.SharedCollection[ClientInterfacer] //map[uint64]ClientInterfacer

	// 广播频道
	BroadcastChan chan *packets.Packet

	//客户端注册渠道
	RegisterChan chan ClientInterfacer

	//反注册渠道
	UnregisterChan chan ClientInterfacer
}

// NewHub returns a new Hub
//
// Hub is a center of the server, it's responsible for manage all the clients and
// broadcast the message to all the clients.
//
// The Hub has four channels:
// - BroadcastChan is a channel for broadcast the message to all the clients.
// - RegisterChan is a channel for register the client to the hub.
// - UnregisterChan is a channel for unregister the client from the hub.
//
// The Hub has a map Client, it's used to store all the clients, the key is the
// client's id, the value is the client's interface.
//
// The Hub is run in a goroutine, it will start a loop to listen the channels and
// broadcast the message to all the clients.
//
// The Hub is the heart of the server, it's responsible for manage all the
// clients and broadcast the message to all the clients.
func NewHub() *Hub {
	return &Hub{
		Clients:        objects.NewSharedCollection[ClientInterfacer](), //make(map[uint64]ClientInterfacer),
		BroadcastChan:  make(chan *packets.Packet),
		RegisterChan:   make(chan ClientInterfacer),
		UnregisterChan: make(chan ClientInterfacer),
	}
}

func (h *Hub) Run() {
	log.Println("Awaiting for connections...")

	for {
		select {
		case client := <-h.RegisterChan:
			//client.Initialize(uint64(len(h.Clients)))
			client.Initialize(h.Clients.Add(client))
		case client := <-h.UnregisterChan:
			// h.Clients[client.Id()] = nil
			h.Clients.Remove(client.Id())
		case packet := <-h.BroadcastChan:
			// for id, client := range h.Clients {
			// 	if id != packet.SenderId {
			// 		client.ProcessMessage(packet.SenderId, packet.Msg)
			// 	}
			// }
			h.Clients.ForEach(func(clientId uint64, client ClientInterfacer) {
				if clientId != packet.SenderId {
					client.ProcessMessage(packet.SenderId, packet.Msg)
				}
			})
		}
	}
}

func (h *Hub) Server(getNewClient func(*Hub, http.ResponseWriter, *http.Request) (ClientInterfacer, error), writer http.ResponseWriter, request *http.Request) {
	log.Println("New client connected from", request.RemoteAddr)
	client, err := getNewClient(h, writer, request)
	if err != nil {
		log.Printf("Error obtaining client:%v", err)
	}

	h.RegisterChan <- client

	go client.WritePump()
	go client.ReadPump()

}
