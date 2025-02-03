package packets

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
