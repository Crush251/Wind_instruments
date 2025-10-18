package main

////////////////////////////////////////////////////////////////////////////////
// 常量定义
////////////////////////////////////////////////////////////////////////////////

const (
	OpCode = 0x01 // CAN数据帧操作码
)

// 手指到数组索引的映射
var fingerIndex = map[string]int{
	"Thumb":          0, // 拇指
	"Thumb rotation": 1, // 拇指旋转
	"Index":          2, // 食指
	"Middle":         3, // 中指
	"Ring":           4, // 无名指
	"Little":         5, // 小指
	"Pinky":          5, // 小指别名
	"Thumb1":         0, // 倍高音拇指（唢呐）
	"Thumb2":         1, // 高音拇指（唢呐）
}

// 全局演奏控制器
var playbackController = &PlaybackController{
	stopChan:   make(chan bool, 1),
	doneChan:   make(chan bool, 1),
	instrument: "sks", // 默认为萨克斯
}
