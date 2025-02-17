extends Node

const packets := preload("res://packets/packets.gd")
const Actor := preload("res://objects/actor/actor.gd")
const Spore := preload("res://objects/spore/spore.gd") #孢子的脚本

@onready var _line_edit: LineEdit = $UI/LineEdit
@onready var _log: Log = $UI/Log

@onready var _world: Node2D = $World

var _players:= {}
var _spores:= {} #孢子的字典

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
	elif packet.has_spore():  #收到孢子信息
		_handle_spore_msg(sender_id, packet.get_spore())
	elif packet.has_spore_consumed(): #收到吃孢子的信息
		_handle_spore_consumed_msg(sender_id, packet.get_spore_consumed())
		
		
func _handle_chat_msg(sender_id:int,chat_msg:packets.ChatMessage) ->void:
	if sender_id in _players:
		var actor := _players[sender_id] as Actor
		_log.chat(actor.actor_name, chat_msg.get_msg())


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
	
	if actor_id not in _players :
		# This is a new player, so we need to create a new actor
		var actor := Actor.instantiate(actor_id, actor_name, x, y, radius, speed, is_player)
		_world.add_child(actor)
		_players[actor_id] = actor
	else:
		var actor2 := _players[actor_id] as Actor
		actor2.position.x = x
		actor2.position.y = y
		
		var direction := player_msg.get_direction()
		actor2.velocity = speed * Vector2.from_angle(direction)
		
func _handle_spore_msg(sender_id: int, spore_msg: packets.SporeMessage) -> void:
	var spore_id := spore_msg.get_id()
	var x := spore_msg.get_x()
	var y := spore_msg.get_y()
	var radius := spore_msg.get_radius()
	
	if spore_id not in _spores:
		var spore := Spore.instantiate(spore_id, x, y, radius)
		_world.add_child(spore)
		_spores[spore_id] = spore

func _add_actor(actor_id: int, actor_name: String, x: float, y: float, radius: float, speed: float, is_player: bool) -> void:
	var actor := Actor.instantiate(actor_id, actor_name, x, y, radius, speed, is_player)
	_world.add_child(actor)
	_players[actor_id] = actor as Actor
	
	if is_player: 
		actor.area_entered.connect(_on_player_area_entered)
		
func _update_actor(actor_id: int, x: float, y: float, direction: float, speed: float, radius: float, is_player: bool) -> void:
	var actor := _players[actor_id] as Actor 
	actor.radius = radius
	
	if actor.position.distance_squared_to(Vector2(x,y)) > 100 :
		actor.position.x = x
		actor.position.y = y 
	
	if not is_player: 
		actor.velocity = Vector2.from_angle(direction) * speed
	
func _on_player_area_entered(area: Area2D) -> void:
	if area is Spore:
		_consume_spore(area as Spore)
	elif area is Actor:
		_collide_actor(area as Actor)
		
func _consume_spore(spore: Spore) -> void:
	#吃下一个孢子后自己成长
	var player = _players[GameManager.client_id]
	var player_mass := _rad_to_mass(player.radius)
	var spore_mass := _rad_to_mass(spore.radius)
	_set_actor_mass(player, player_mass + spore_mass)
	
	var packet := packets.Packet.new()
	var spore_consumed_msg := packet.new_spore_consumed()
	spore_consumed_msg.set_spore_id(spore.spore_id)
	WS.send(packet)
	_remove_spore(spore)

func _collide_actor(actor: Actor) -> void:
	var player := _players[GameManager.client_id] as Actor
	var player_mass := _rad_to_mass(player.radius)
	var actor_mass := _rad_to_mass(actor.radius)
	
	if player_mass > actor_mass * 1.5:
		_consume_actor(actor)

#删除孢子
func _remove_spore(spore: Spore) -> void:
	_spores.erase(spore.spore_id)
	spore.queue_free()

#删除角色
func _remove_actor(actor: Actor) -> void:
	_players.erase(actor.actor_id)
	actor.queue_free()

func _handle_spore_consumed_msg(sender_id: int, spore_consumed_msg: packets.SporeConsumedMessage) -> void:
	if sender_id in _players:
		var actor := _players[sender_id] as Actor
		var actor_mass := _rad_to_mass(actor.radius)
		
		var spore_id := spore_consumed_msg.get_spore_id()
		if spore_id in _spores:
			var spore := _spores[spore_id] as Spore
			var spore_mass := _rad_to_mass(spore.radius)
			_set_actor_mass(actor, actor_mass + spore_mass)
			_remove_spore(spore) #删除孢子
			

func _rad_to_mass(radius: float) -> float:
	return radius * radius * PI

#设置新的大小
func _set_actor_mass(actor: Actor, new_mass: float) -> void:
	actor.radius = sqrt(new_mass / PI)
	
func _consume_actor(actor: Actor) -> void:
	var player := _players[GameManager.client_id] as Actor
	var player_mass := _rad_to_mass(player.radius)
	var actor_mass := _rad_to_mass(actor.radius)
	_set_actor_mass(player, player_mass + actor_mass)
	
	var packet := packets.Packet.new()
	var player_consumed_msg := packet.new_player_consumed()
	player_consumed_msg.set_player_id(actor.actor_id)
	WS.send(packet)
	_remove_actor(actor)
