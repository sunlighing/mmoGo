package packets

import "server/internal/server/objects"

type Msg = isPacket_Msg

func NewChat(text string) Msg {
	return &Packet_Chat{
		Chat: &ChatMessage{
			Msg: text,
		},
	}
}

// NewId creates a new packet with the type "id" and the given id number.
//
// The returned packet is a pointer to a Packet_Id struct.
func NewId(id uint64) Msg {
	return &Packet_Id{
		Id: &IdMessage{
			Id: id,
		},
	}
}

func NewDenyResponse(reason string) Msg {
	return &Packet_DenyResponse{
		DenyResponse: &DenyResponseMessage{
			Reason: reason,
		},
	}
}

func NewOkResponse() Msg {
	return &Packet_OkResponse{
		OkResponse: &OkResponseMessage{},
	}
}

//构建新的玩家的信息
func NewPlayer(id uint64, player *objects.Player) Msg {
	return &Packet_Player{
		Player: &PlayerMessage{
			Id:        id,
			Name:      player.Name,
			X:         player.X,
			Y:         player.Y,
			Radius:    player.Radius,
			Direction: player.Direction,
			Speed:     player.Speed,
		},
	}
}

//生成一个孢子
func NewSpore(id uint64, spore *objects.Spore) Msg {
	return &Packet_Spore{
		Spore: &SporeMessage{
			Id:     id,
			X:      spore.X,
			Y:      spore.Y,
			Radius: spore.Radius,
		},
	}
}
