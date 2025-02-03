package clients

import (
	"fmt"
	"log"
	"net/http"

	"server/internal/server"
	"server/pkg/packets"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type WebSocketClient struct {
	id       uint64
	conn     *websocket.Conn
	hub      *server.Hub
	sendChan chan *packets.Packet
	logger   *log.Logger
}

func NewWebSocketClient(hub *server.Hub, writer *http.ResponseWriter, requst *http.Request) (server.ClientInterfacer, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	//可以禁止默认连接

	conn, err := upgrader.Upgrade(*writer, requst, nil)

	if err != nil {
		return nil, err
	}

	c := &WebSocketClient{
		hub:      hub,
		conn:     conn,
		sendChan: make(chan *packets.Packet, 256),
		logger:   log.New(log.Writer(), "Client unknown", log.LstdFlags),
	}

	return c, nil
}

func (c *WebSocketClient) Id() uint64 {
	return c.id
}

func (c *WebSocketClient) ProcessMessage(senderId uint64, messgae packets.Msg) {

}

func (c *WebSocketClient) SocketSend(msg packets.Msg) {

}

func (c *WebSocketClient) PassToPeer(message packets.Msg, peerId uint64) {
	if peer, exists := c.hub.Clients[peerId]; exists {
		peer.ProcessMessage(c.id, message)
	}
}

// PassToPeer is a method for passing a message to another peer.
// peerId is the id of the peer that the message should be sent to.
// The method is used by the server to send messages to other peers.

func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.logger.Panicln("Close read pump")
		c.Close("read pump closed")
	}()

	for {
		//连接上读取消息
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			//异常关闭
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Panicf("Error %v", err)
			}
			break
		}

		//解包
		packet := &packets.Packet{}
		err = proto.Unmarshal(data, packet)
		if err != nil {
			c.logger.Printf("error unmarshalling data: %v", err)
			continue
		}

		//如果客户端没有发包过来，就默认是自己的ID
		if packet.SenderId == 0 {
			packet.SenderId = c.id
		}

		c.ProcessMessage(packet.SenderId, packet.Msg)
	}
}

func (c *WebSocketClient) WritePump() {
	defer func() {
		c.logger.Panicln("Closing Write Pump")
		c.Close("write pump closed")
	}()

	for packet := range c.sendChan {

		// 获得缓存中的读取
		writer, err := c.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			c.logger.Printf("error getting writer for %T packet,closing client: %v", packet.Msg, err)
		}

		//做成字节
		data, err := proto.Marshal(packet)
		if err != nil {
			c.logger.Printf("error marshalling %T packet,closing client:%v", packet.Msg, err)
			continue
		}

		_, err = writer.Write(data)

		if err != nil {
			c.logger.Printf("error writing %T packet: %v", packet.Msg, err)
		}

		writer.Write([]byte{'\n'}) //单引号和双引号的区别是啥

		if err = writer.Close(); err != nil {
			c.logger.Printf("error closing writer for %T packet %v", packet.Msg, err)
			continue
		}
	}
}

func (c *WebSocketClient) Close(reson string) {

}

func (c *WebSocketClient) Initialize(id uint64) {
	c.id = id
	c.logger.SetPrefix(fmt.Sprintf("Client %d", c.id))
}

func (c *WebSocketClient) SocketSendAs(message packets.Msg, senderId uint64) {
	select {
	case c.sendChan <- &packets.Packet{SenderId: senderId, Msg: message}:
	default:
		c.logger.Panicln("send channel full,droping message *T", message)
	}
}

func (c *WebSocketClient) Broadcast(message packets.Msg) {
	c.hub.BroadcastChan <- &packets.Packet{SenderId: c.id, Msg: message}
}
