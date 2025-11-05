package main

import (
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// 执行序列相关数据结构
////////////////////////////////////////////////////////////////////////////////

// ExecutionSequence 预计算的执行序列
type ExecutionSequence struct {
	Meta   SequenceMeta     `json:"meta"`
	Events []ExecutionEvent `json:"events"`
}

// SequenceMeta 执行序列元数据
type SequenceMeta struct {
	SourceFile      string    `json:"source_file"`       // 源音乐文件
	Instrument      string    `json:"instrument"`        // 乐器类型
	BPM             float64   `json:"bpm"`               // BPM
	TonguingDelay   int       `json:"tonguing_delay_ms"` // 吐音延迟（毫秒）
	TotalDurationMS float64   `json:"total_duration_ms"` // 总时长（毫秒）
	TotalEvents     int       `json:"total_events"`      // 事件总数
	GeneratedAt     time.Time `json:"generated_at"`      // 生成时间
	Version         string    `json:"version"`           // 版本号
}

// ExecutionEvent 执行事件（简化版）
type ExecutionEvent struct {
	TimestampMS float64        `json:"t"`                // 绝对时间戳（毫秒）
	DurationMS  float64        `json:"d"`                // 持续时长（毫秒）
	Note        string         `json:"n"`                // 音符名称（调试用）
	Frames      []ExecCANFrame `json:"frames,omitempty"` // CAN帧数组（为空时省略）
	SerialCmd   string         `json:"serial,omitempty"` // 串口命令（"on"/"off"）
}

// ExecCANFrame 执行用CAN帧（简化版）
type ExecCANFrame struct {
	Hand string `json:"hand"` // 手部标识：left/right（逻辑标识，执行时映射到实际接口）
	ID   string `json:"id"`   // 设备ID（如"0x28"）
	Data []byte `json:"d"`    // 数据字节数组
}
