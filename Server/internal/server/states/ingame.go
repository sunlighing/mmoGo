package states

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"server/internal/server"
	"server/internal/server/objects"
	"server/pkg/packets"
	"time"
)

type InGame struct {
	client                 server.ClientInterfacer
	player                 *objects.Player
	logger                 *log.Logger
	cancelPlayerUpdateLoop context.CancelFunc
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

	// Set the initial properties of the player
	g.player.X, g.player.Y = objects.SpawnCoords(g.player.Radius, g.client.SharedGameObjects().Players, nil)

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

	// 进入到游戏后每个五秒钟就发送
	go func() {
		g.client.SharedGameObjects().Spores.ForEach(func(sporeId uint64, spore *objects.Spore) {
			time.Sleep(5 * time.Millisecond)
			g.client.SocketSend(packets.NewSpore(sporeId, spore))
		})
	}()
}

func (g *InGame) HandlerMessage(senderId uint64, message packets.Msg) {
	switch message := message.(type) {
	case *packets.Packet_Player:
		g.handlePlayer(senderId, message)
	case *packets.Packet_PlayerDirection:
		g.handlePlayerDirection(senderId, message)
	case *packets.Packet_Chat:
		g.handleChat(senderId, message)
	case *packets.Packet_SporeConsumed:
		g.logger.Printf("Spore %d consumed by client %d", message.SporeConsumed.SporeId, senderId)
		g.handleSporeConsumed(senderId, message) //处理孢子被吃的事件
	case *packets.Packet_PlayerConsumed:
		g.handlePlayerConsumed(senderId, message)
	case *packets.Packet_Spore:
		g.handleSpore(senderId, message)
	}
}

func (g *InGame) OnExit() {
	//游戏对象里头删除这个对象
	if g.cancelPlayerUpdateLoop != nil {
		g.cancelPlayerUpdateLoop()
	}
	g.client.SharedGameObjects().Players.Remove(g.client.Id())
}

func (g *InGame) syncPlayer(delta float64) {
	newX := g.player.X + g.player.Speed*math.Cos(g.player.Direction)*delta
	newY := g.player.Y + g.player.Speed*math.Sin(g.player.Direction)*delta

	g.player.X = newX
	g.player.Y = newY

	updatePacket := packets.NewPlayer(g.client.Id(), g.player)
	g.client.Broadcast(updatePacket)
	go g.client.SocketSend(updatePacket)
}

func (g *InGame) handlePlayer(senderId uint64, message *packets.Packet_Player) {
	//如果是自己的情况就转发
	if senderId == g.client.Id() {
		g.logger.Println("Received player message from our own client, ignoring")
		return
	}
	g.client.SocketSendAs(message, senderId)
}

func (g *InGame) playerUpdateLoop(ctx context.Context) {
	const delta float64 = 0.05
	ticker := time.NewTicker(time.Duration(delta*1000) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.syncPlayer(delta)
		case <-ctx.Done():
			return
		}
	}
}

func (g *InGame) handlePlayerDirection(senderId uint64, message *packets.Packet_PlayerDirection) {
	if senderId == g.client.Id() {
		g.player.Direction = message.PlayerDirection.Direction

		// If this is the first time receiving a player direction message from our client, start the player update loop
		if g.cancelPlayerUpdateLoop == nil {
			ctx, cancel := context.WithCancel(context.Background())
			g.cancelPlayerUpdateLoop = cancel
			go g.playerUpdateLoop(ctx)
		}
	}
}

// 游戏内聊天
func (g *InGame) handleChat(senderId uint64, message *packets.Packet_Chat) {
	if senderId == g.client.Id() {
		g.client.Broadcast(message)
	} else {
		g.client.SocketSendAs(message, senderId)
	}
}

// 处理孢子被吃的事件
func (g *InGame) handleSporeConsumed(senderId uint64, message *packets.Packet_SporeConsumed) {
	// We will implement this method in a moment
	if senderId != g.client.Id() {
		g.client.SocketSendAs(message, senderId)
		return
	}

	// If the spore was supposedly consumed by our own player, we need to verify the plausibility of the event
	errMsg := "Could not verify spore consumption: "

	// First check if the spore exists
	sporeId := message.SporeConsumed.SporeId
	spore, err := g.getSpore(sporeId)
	if err != nil {
		g.logger.Println(errMsg + err.Error())
		return
	}

	// Next, check if the spore is close enough to the player to be consumed
	err = g.validatePlayerCloseToObject(spore.X, spore.Y, spore.Radius, 10)
	if err != nil {
		g.logger.Println(errMsg + err.Error())
		return
	}

	// If we made it this far, the spore consumption is valid, so grow the player, remove the spore, and broadcast the event
	sporeMass := radToMass(spore.Radius)
	g.player.Radius = g.nextRadius(sporeMass)

	go g.client.SharedGameObjects().Spores.Remove(sporeId)

	g.client.Broadcast(message)
}

// 判断孢子是否存在
func (g *InGame) getSpore(sporeId uint64) (*objects.Spore, error) {
	spore, exists := g.client.SharedGameObjects().Spores.Get(sporeId)
	if !exists {
		return nil, fmt.Errorf("spore with ID %d does not exist", sporeId)
	}
	return spore, nil
}

// 判断孢子是否在附近
func (g *InGame) validatePlayerCloseToObject(objX, objY, objRadius, buffer float64) error {
	realDX := g.player.X - objX
	realDY := g.player.Y - objY
	realDistSq := realDX*realDX + realDY*realDY

	thresholdDist := g.player.Radius + buffer + objRadius
	thresholdDistSq := thresholdDist * thresholdDist

	if realDistSq > thresholdDistSq {
		return fmt.Errorf("player is too far from the object (distSq: %f, thresholdSq: %f)", realDistSq, thresholdDistSq)
	}
	return nil
}

// 计算圆的面积
func radToMass(radius float64) float64 {
	return math.Pi * radius * radius
}

func massToRad(mass float64) float64 {
	return math.Sqrt(mass / math.Pi)
}

// 计算吃下孢子的体积
func (g *InGame) nextRadius(massDiff float64) float64 {
	oldMass := radToMass(g.player.Radius)
	newMass := oldMass + massDiff
	return massToRad(newMass)
}

// 吞并玩家
func (g *InGame) handlePlayerConsumed(senderId uint64, message *packets.Packet_PlayerConsumed) {
	if senderId != g.client.Id() {
		g.client.SocketSendAs(message, senderId)

		if message.PlayerConsumed.PlayerId == g.client.Id() {
			log.Println("Player was consumed, respawning")
			g.client.SetState(&InGame{
				player: &objects.Player{
					Name: g.player.Name,
				},
			})
		}

		return
	}

	// If the other player was supposedly consumed by our own player, we need to verify the plausibility of the event
	errMsg := "Could not verify player consumption: "

	// First, check if the player exists
	otherId := message.PlayerConsumed.PlayerId
	other, err := g.getOtherPlayer(otherId)
	if err != nil {
		g.logger.Println(errMsg + err.Error())
		return
	}

	// Next, check the other player's mass is smaller than our player's
	ourMass := radToMass(g.player.Radius)
	otherMass := radToMass(other.Radius)
	if ourMass <= otherMass*1.5 {
		g.logger.Printf(errMsg+"player not massive enough to consume the other player (our radius: %f, other radius: %f)", g.player.Radius, other.Radius)
		return
	}

	// Finally, check if the player is close enough to the other to be consumed
	err = g.validatePlayerCloseToObject(other.X, other.Y, other.Radius, 10)
	if err != nil {
		g.logger.Println(errMsg + err.Error())
		return
	}

	// If we made it this far, the player consumption is valid, so grow the player, remove the consumed other, and broadcast the event
	g.player.Radius = g.nextRadius(otherMass)

	go g.client.SharedGameObjects().Players.Remove(otherId)

	g.client.Broadcast(message)
}

// 判断是否有这个玩家
func (g *InGame) getOtherPlayer(otherId uint64) (*objects.Player, error) {
	other, exists := g.client.SharedGameObjects().Players.Get(otherId)
	if !exists {
		return nil, fmt.Errorf("player with ID %d does not exist", otherId)
	}
	return other, nil
}

// 转发孢子的消息
func (g *InGame) handleSpore(senderId uint64, message *packets.Packet_Spore) {
	g.client.SocketSendAs(message, senderId)
}
