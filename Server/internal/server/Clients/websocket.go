package clients

import (
	"fmt"
	"log"
	"net/http"

	"server/internal/server"
	"server/internal/server/states"
	"server/pkg/packets"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type WebSocketClient struct {
	id       uint64
	conn     *websocket.Conn
	hub      *server.Hub
	sendChan chan *packets.Packet
	state    server.ClientStateHandler
	logger   *log.Logger
	dbTx     *server.DbTx
}

func NewWebSocketClient(hub *server.Hub, writer http.ResponseWriter, requst *http.Request) (server.ClientInterfacer, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	//可以禁止默认连接

	conn, err := upgrader.Upgrade(writer, requst, nil)

	if err != nil {
		return nil, err
	}

	c := &WebSocketClient{
		hub:      hub,
		conn:     conn,
		sendChan: make(chan *packets.Packet, 256),
		logger:   log.New(log.Writer(), "Client unknown", log.LstdFlags),
		dbTx:     hub.NewDbTx(),
	}

	return c, nil
}

func (c *WebSocketClient) Id() uint64 {
	return c.id
}

func (c *WebSocketClient) SetState(state server.ClientStateHandler) {
	prevStateName := "None"

	if c.state != nil {
		prevStateName = c.state.Name()
		c.state.OnExit()
	}

	newStateName := "None"

	if state != nil {
		newStateName = state.Name()
	}

	c.logger.Printf("Switching from state %s to %s", prevStateName, newStateName)

	c.state = state

	if c.state != nil {
		c.state.SetClient(c)
		c.state.OnEnter()
	}
}

func (c *WebSocketClient) ProcessMessage(senderId uint64, message packets.Msg) {
	// c.logger.Printf("Received message: %T from client - echoing back ...", messgae)
	// c.SocketSend(messgae)

	//如果是自己就广播给别人
	c.state.HandlerMessage(senderId, message)

}

func (c *WebSocketClient) SocketSend(message packets.Msg) {
	c.SocketSendAs(message, c.id)
}

func (c *WebSocketClient) PassToPeer(message packets.Msg, peerId uint64) {
	//c.hub.Clients[peerId]
	if peer, exists := c.hub.Clients.Get(peerId); exists {
		peer.ProcessMessage(c.id, message)
	}
}

// PassToPeer is a method for passing a message to another peer.
// peerId is the id of the peer that the message should be sent to.
// The method is used by the server to send messages to other peers.

func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.logger.Printf("Close read pump")
		c.Close("read pump closed")
	}()

	for {
		//连接上读取消息
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			//异常关闭
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Printf("Error %v", err)
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
	c.logger.Printf("Closing Connection because %s", reson)

	c.SetState(nil)

	c.hub.UnregisterChan <- c
	c.conn.Close()
	if _, closed := <-c.sendChan; !closed {
		close(c.sendChan)
	}
}

func (c *WebSocketClient) Initialize(id uint64) {
	c.id = id
	c.logger.SetPrefix(fmt.Sprintf("ClientID : %d ,", c.id))
	c.SetState(&states.Connected{})
	// c.SocketSend(packets.NewId(c.id))
	c.logger.Printf("Sent ID to Client")
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

func (c *WebSocketClient) DbTx() *server.DbTx {
	return c.dbTx
}
