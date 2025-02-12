package states

import (
	"fmt"
	"log"
	"math/rand/v2"
	"server/internal/server"
	"server/internal/server/objects"
	"server/pkg/packets"
)

type InGame struct {
	client server.ClientInterfacer
	player *objects.Player
	logger *log.Logger
}

func (g *InGame) Name() string {
	return "InGame"
}

//游戏中的状态

func (g *InGame) SetClient(client server.ClientInterfacer) {
	g.client = client
	loggingPrefix := fmt.Sprintf("Client %d [%s] :", client.Id())

	g.logger = log.New(log.Writer(), loggingPrefix, log.LstdFlags)
}

func (g *InGame) OnEnter() {
	log.Printf("Adding player %s to the shared collection", g.player.Name)

	//共享的gameObjects 池子里面，添加玩家的player ID 和 客户端ID
	//里面是加了线程锁的，所以添加游戏对象是在线程里面去添加游戏对象
	go g.client.SharedGameObjects().Players.Add(g.player, g.client.Id())

	// 进入到了游戏后开始设置位置参数
	g.player.X = rand.Float64() * 1000
	g.player.Y = rand.Float64() * 1000
	g.player.Speed = 150.0
	g.player.Radius = 20.0

	// Send the player's initial state to the client
	g.client.SocketSend(packets.NewPlayer(g.client.Id(), g.player))
}

func (g *InGame) HandlerMessage(senderId uint64, message packets.Msg) {

}

func (g *InGame) OnExit() {
	//游戏对象里头删除这个对象
	g.client.SharedGameObjects().Players.Remove(g.client.Id())
}
