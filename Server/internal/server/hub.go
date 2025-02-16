package server

import (
	"context"
	"database/sql"
	"log"
	"math/rand/v2"
	"net/http"
	"server/internal/server/db"
	"server/internal/server/objects"
	"server/pkg/packets"

	_ "embed"

	_ "modernc.org/sqlite"
)

// 最大孢子的数量
const MaxSpores int = 1000

// Embed the database schema to be used when creating the database tables
//
//go:embed db/config/schema.sql
var schemaGenSql string

// A structure for database transaction context
type DbTx struct {
	Ctx     context.Context
	Queries *db.Queries
}

func (h *Hub) NewDbTx() *DbTx {
	return &DbTx{
		Ctx:     context.Background(),
		Queries: db.New(h.dbPool),
	}
}

type SharedGameObjects struct {
	//这个ID 是client id 连接ID
	Players *objects.SharedCollection[*objects.Player]
	//这个是孢子池
	Spores *objects.SharedCollection[*objects.Spore]
}

// 客户端状态机句柄
type ClientStateHandler interface {
	Name() string

	//注入一个客户端的handler
	SetClient(client ClientInterfacer)

	//进入时的执行函数
	OnEnter()

	//消息句柄
	HandlerMessage(senderId uint64, message packets.Msg)

	//退出时的执行函数
	OnExit()
}

type ClientInterfacer interface {
	Initialize(id uint64)

	SetState(newState ClientStateHandler)

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

	// A reference to the database transaction context for this client 管理客户端的DB
	DbTx() *DbTx

	Close(reson string)

	SharedGameObjects() *SharedGameObjects

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

	// Database connection pool db 连接池
	dbPool *sql.DB

	//游戏池的对象
	SharedGameObjects *SharedGameObjects
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

	//定义数据库池
	dbPool, err := sql.Open("sqlite", "db.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	return &Hub{
		Clients:        objects.NewSharedCollection[ClientInterfacer](), //make(map[uint64]ClientInterfacer),
		BroadcastChan:  make(chan *packets.Packet),
		RegisterChan:   make(chan ClientInterfacer),
		UnregisterChan: make(chan ClientInterfacer),
		dbPool:         dbPool,
		SharedGameObjects: &SharedGameObjects{
			Players: objects.NewSharedCollection[*objects.Player](),
			Spores:  objects.NewSharedCollection[*objects.Spore](), //生成一个孢子池的对象
		},
	}
}

func (h *Hub) Run() {
	log.Println("Awaiting for connections...")

	//初始化数据库
	log.Println("Initializing database...")
	if _, err := h.dbPool.ExecContext(context.Background(), schemaGenSql); err != nil {
		log.Fatal(err)
	}

	// 测试用 生成不同的孢子
	log.Println("Placing spores")

	for i := 0; i < MaxSpores; i++ {
		h.SharedGameObjects.Spores.Add(h.NewSpore())
	}

	//等待客户端连接
	log.Println("Awaiting client registraions")

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

// 新建一个孢子
func (h *Hub) NewSpore() *objects.Spore {
	sporeRadius := max(rand.NormFloat64()*3+10, 5)
	x, y := objects.SpawnCoords()
	return &objects.Spore{X: x, Y: y, Radius: sporeRadius}
}
