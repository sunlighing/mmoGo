# mmoGo
使用go和Godot 游戏服务器教学

# proto 编译命令

// protoc -I="shared" --go_out="server" "shared/packets.proto"

//相对文件夹下
.\Tools\protoc\bin\protoc.exe -I="shared" --go_out="server" "shared/packets.proto"