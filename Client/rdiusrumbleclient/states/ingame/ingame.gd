extends Node

const packets := preload("res://packets/packets.gd")
const Actor := preload("res://objects/actor/actor.gd")


@onready var _line_edit: LineEdit = $UI/LineEdit
@onready var _log: Log = $UI/Log

@onready var _world: Node2D = $World

func _ready() -> void:
	WS.connection_closed.connect(_on_ws_connection_closed)
	WS.packet_received.connect(_on_ws_packet_received)
	
	_line_edit.text_submitted.connect(_on_line_edit_text_submitted)
	

func _on_ws_connection_closed() -> void:
	_log.warning("Connection Closed")
	
func _on_ws_packet_received(packet: packets.Packet) -> void:
	var sender_id := packet.get_sender_id() 
	if packet.has_chat():
		_handle_chat_msg(sender_id,packet.get_chat())
	elif packet.has_player():
		_handle_player_msg(sender_id, packet.get_player())
		
func _handle_chat_msg(sender_id:int,chat_msg:packets.ChatMessage) ->void:
	_log.chat("Client %d" % sender_id,chat_msg.get_msg())


#参数是输入框内消息
func _on_line_edit_text_submitted(new_text:String) -> void:
	var packet := packets.Packet.new()
	var chat_msg := packet.new_chat()
	chat_msg.set_msg(new_text)
	
	var err := WS.send(packet)
	
	if err:
		_log.error("Error sending chat message")
	else:
		_log.chat("You",new_text)
	
	_line_edit.clear()
	
#收到玩家进入的消息
func _handle_player_msg(sender_id: int, player_msg: packets.PlayerMessage) -> void:
	var actor_id := player_msg.get_id()
	var actor_name := player_msg.get_name()
	var x := player_msg.get_x()
	var y := player_msg.get_y()
	var radius := player_msg.get_radius()
	var speed := player_msg.get_speed()

	var is_player := actor_id == GameManager.client_id
	
	var actor := Actor.instantiate(actor_id, actor_name, x, y, radius, speed, is_player)
	_world.add_child(actor)
