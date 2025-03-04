package objects

// 构建玩家的基本要素

type Player struct {
	Name      string
	X         float64 //坐标X
	Y         float64 //坐标Y
	Radius    float64 //范围
	Direction float64 //方向
	Speed     float64 //速度
}

// 构成孢子的基本要素
type Spore struct {
	X      float64
	Y      float64
	Radius float64
}
