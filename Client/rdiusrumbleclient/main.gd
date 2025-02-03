extends Node

const packets := preload("res://packets/packets.gd")

@onready var _log := $Log

func _ready() -> void:
	WS.connected_to_server.connect(_on_ws_connected_to_server)
	WS.connection_closed.connect(_on_ws_connection_closed)
	WS.packet_received.connect(_on_ws_packet_received)
	
	print("Connect to server...")
	WS.connect_to_url("ws://localhost:8080/ws")
	
func _on_ws_connected_to_server() -> void:
	#测试，发送一个小希
	var packet := packets.Packet.new()
	var chat_msg := packet.new_chat()
	chat_msg.set_msg("Hello,from godot")
	
	var err := WS.send(packet)
	
	if err:
		print("Error sending packet")
	else:
		print("Packet sent successfully!")

func _on_ws_connection_closed() -> void:
	print("Connection Closed")
	
func _on_ws_packet_received(packet: packets.Packet) -> void:
	print("Received packet from the server %s" % packet)
	
