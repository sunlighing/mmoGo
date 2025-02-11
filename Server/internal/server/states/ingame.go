package states

import (
	"fmt"
	"log"
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
}

func (g *InGame) HandlerMessage(senderId uint64, message packets.Msg) {

}

func (g *InGame) OnExit() {
	//游戏对象里头删除这个对象
	g.client.SharedGameObjects().Players.Remove(g.client.Id())
}
