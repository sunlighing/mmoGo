extends Node

enum State {
	ENTERED,
	INGAME,
}
#[State,String] 
var _states_scene:Dictionary ={
	State.ENTERED:"res://states/entered/entered.tscn",
	State.INGAME:"res://states/ingame/ingame.tscn",
}

var client_id:int 
var _current_scene_root:Node 

func set_state(state:State) -> void: 
	#释放当前状态的节点
	if _current_scene_root != null:
		_current_scene_root.queue_free()
		
	#预加载场景中的节点
	var scene: PackedScene = load(_states_scene[state])  #记得判断下有没有当前场景
	_current_scene_root = scene.instantiate() #实例化场景
	add_child(_current_scene_root)
