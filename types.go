package main

import (
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// 配置与数据结构定义
////////////////////////////////////////////////////////////////////////////////

// 演奏配置
type Config struct {
	JsonPath      string  `yaml:"json_path"`      // 音乐时间轴JSON文件路径
	FingeringYAML string  `yaml:"fingering_yaml"` // 指法映射YAML文件路径
	BPM           float64 `yaml:"bpm"`            // 节拍速度（每分钟节拍数）
	CanBridgeURL  string  `yaml:"can_bridge_url"` // CAN总线桥接服务地址
	DryRun        bool    `yaml:"dry_run"`        // 是否为调试模式（只打印不发送）

	Hands struct {
		Left  HandConfig `yaml:"left"`  // 左手配置
		Right HandConfig `yaml:"right"` // 右手配置
	} `yaml:"hands"`
	QibengInterface string `yaml:"qibenginterface"`

	// 气泵控制配置（仅串口）
	Pump struct {
		PortName string `yaml:"port_name"` // 串口名称（如：/dev/ttyUSB0）
	} `yaml:"pump"`

	// 萨克斯手指力度配置：[拇指, 拇指旋转, 食指, 中指, 无名指, 小指]
	SksLeftPressProfile    []int `yaml:"sks_left_press_profile"`    // 萨克斯左手按压力度
	SksLeftReleaseProfile  []int `yaml:"sks_left_release_profile"`  // 萨克斯左手释放力度
	SksRightPressProfile   []int `yaml:"sks_right_press_profile"`   // 萨克斯右手按压力度
	SksRightReleaseProfile []int `yaml:"sks_right_release_profile"` // 萨克斯右手释放力度

	// 唢呐手指力度配置：[拇指, 拇指旋转, 食指, 中指, 无名指, 小指]
	SnLeftPressProfile    []int `yaml:"sn_left_press_profile"`    // 唢呐左手按压力度
	SnLeftReleaseProfile  []int `yaml:"sn_left_release_profile"`  // 唢呐左手释放力度
	SnRightPressProfile   []int `yaml:"sn_right_press_profile"`   // 唢呐右手按压力度
	SnRightReleaseProfile []int `yaml:"sn_right_release_profile"` // 唢呐右手释放力度

	// 唢呐高音和倍高音配置
	SnLeftHighThumb    []int `yaml:"sn_left_high_Thumb"`     // 唢呐高音Thumb2配置
	SnLeftHighProThumb []int `yaml:"sn_left_high_pro_Thumb"` // 唢呐倍高音Thumb1配置

	Ready struct {
		Enabled bool `yaml:"enabled"` // 是否启用预备手势
		HoldMS  int  `yaml:"hold_ms"` // 预备手势持续时间（毫秒）
	} `yaml:"ready"`
}

// 手部配置
type HandConfig struct {
	Interface string `yaml:"interface"` // CAN接口名称（can0/can1等）
	ID        string `yaml:"id"`        // CAN设备ID
}

// 时间轴文件结构
type TimelineFile struct {
	Meta     map[string]any `json:"meta"`     // 元数据（包含BPM等信息）
	Timeline [][]any        `json:"timeline"` // 时间轴：[[音符, 持续拍数], ...]
}

// 指法映射条目
type FingeringEntry struct {
	Note  string   `yaml:"note"`  // 音符（如"A4"）
	Left  []string `yaml:"left"`  // 左手需要按下的手指
	Right []string `yaml:"right"` // 右手需要按下的手指
}

// 指法配置
type FingeringConfig struct {
	FingeringMap []FingeringEntry `yaml:"fingering_map"`
}

////////////////////////////////////////////////////////////////////////////////
// Web服务相关结构体
////////////////////////////////////////////////////////////////////////////////

// 音乐文件信息
type MusicFileInfo struct {
	Filename   string  `json:"filename"`    // 文件名
	Title      string  `json:"title"`       // 曲目标题
	BPM        float64 `json:"bpm"`         // 节拍速度
	Duration   int     `json:"duration"`    // 时长（音符数量）
	FilePath   string  `json:"file_path"`   // 完整文件路径
	FileSize   int64   `json:"file_size"`   // 文件大小
	ModifiedAt string  `json:"modified_at"` // 修改时间
}

// 演奏状态
type PlaybackStatus struct {
	IsPlaying           bool                 `json:"is_playing"`           // 是否正在演奏
	CurrentFile         string               `json:"current_file"`         // 当前文件
	CurrentNote         int                  `json:"current_note"`         // 当前音符索引
	TotalNotes          int                  `json:"total_notes"`          // 总音符数
	ElapsedTime         string               `json:"elapsed_time"`         // 已播放时间
	RemainingTime       string               `json:"remaining_time"`       // 剩余时间
	Progress            float64              `json:"progress"`             // 播放进度（0-100）
	TheoreticalDuration float64              `json:"theoretical_duration"` // 理论时长（秒）
	ActualDuration      float64              `json:"actual_duration"`      // 实际时长（秒）
	SignificantRests    []RestTimingResponse `json:"significant_rests"`    // 显著空拍列表
}

// RestTimingResponse 空拍时间响应（用于前端显示）
type RestTimingResponse struct {
	StartOffset float64 `json:"start_offset"` // 起始偏移（秒）
	EndOffset   float64 `json:"end_offset"`   // 结束偏移（秒）
	Duration    float64 `json:"duration"`     // 持续时长（秒）
	Beats       float64 `json:"beats"`        // 拍数
}

// 演奏控制器
type PlaybackController struct {
	mutex        sync.RWMutex
	status       PlaybackStatus
	stopChan     chan bool
	doneChan     chan bool // 播放完成信号
	isRunning    bool
	config       Config
	timeline     TimelineFile
	fingeringMap map[string]FingeringEntry
	startTime    time.Time
	instrument   string // "sks" 或 "sn"，表示当前乐器类型
}

////////////////////////////////////////////////////////////////////////////////
// 演奏引擎相关结构体
////////////////////////////////////////////////////////////////////////////////

// 音符事件结构
type NoteEvent struct {
	Note     string
	Duration float64
	Index    int
}

// 演奏引擎
type PerformanceEngine struct {
	cfg            Config
	fingeringMap   map[string]FingeringEntry
	instrument     string
	secondsPerBeat float64
	lastThumbState string       // 追踪上一个音符的拇指状态：""、"Thumb1"、"Thumb2"
	timeline       TimelineFile // 时间轴数据
	tonguingDelay  int          // 吐音延迟时间（毫秒）
}

////////////////////////////////////////////////////////////////////////////////
// CAN通信相关结构体
////////////////////////////////////////////////////////////////////////////////

// CAN消息结构
type CanMessage struct {
	Interface string `json:"interface"` // CAN接口
	Id        uint32 `json:"id"`        // CAN设备ID
	Data      []byte `json:"data"`      // 数据内容
}
