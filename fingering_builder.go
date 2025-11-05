package main

import (
	"strings"
)

////////////////////////////////////////////////////////////////////////////////
// 指法构建器模块
////////////////////////////////////////////////////////////////////////////////

// FingeringBuilder 指法构建器
type FingeringBuilder struct{}

// NewFingeringBuilder 创建新的指法构建器
func NewFingeringBuilder() *FingeringBuilder {
	return &FingeringBuilder{}
}

// BuildFingerFrame 构建手指动作的CAN数据帧（支持乐器类型和高音处理）
// 参数：pressedFingers - 需要按下的手指列表
//
//	pressProfile - 按压力度配置
//	releaseProfile - 释放力度配置
//	cfg - 配置信息（用于唢呐高音处理）
//	instrument - 乐器类型（"sks"或"sn"）
func (fb *FingeringBuilder) BuildFingerFrame(pressedFingers []string, pressProfile []int, releaseProfile []int, cfg Config, instrument string) []byte {
	frame := make([]byte, 7)
	frame[0] = OpCode

	// 初始化所有手指为释放状态
	fb.setDefaultReleaseValues(frame, releaseProfile)

	// 根据乐器类型处理
	if instrument == "sn" {
		fb.buildSuonaFrame(frame, pressedFingers, pressProfile, cfg)
	} else {
		fb.buildSaxophoneFrame(frame, pressedFingers, pressProfile)
	}

	return frame
}

// SetDefaultReleaseValues 设置默认释放值
func (fb *FingeringBuilder) setDefaultReleaseValues(frame []byte, releaseProfile []int) {
	for i := 0; i < 6; i++ {
		if i < len(releaseProfile) {
			frame[i+1] = byte(releaseProfile[i])
		} else {
			frame[i+1] = 255
		}
	}
}

// BuildSuonaFrame 构建唢呐指法帧
func (fb *FingeringBuilder) buildSuonaFrame(frame []byte, pressedFingers []string, pressProfile []int, cfg Config) {
	// 检查高音拇指类型
	thumbType := fb.getSuonaThumbType(pressedFingers)

	// 处理特殊拇指配置
	switch thumbType {
	case "Thumb1": // 倍高音
		if len(cfg.SnLeftHighProThumb) >= 2 {
			frame[1] = byte(cfg.SnLeftHighProThumb[0])
			frame[2] = byte(cfg.SnLeftHighProThumb[1])
		}
	case "Thumb2": // 高音
		if len(cfg.SnLeftHighThumb) >= 2 {
			frame[1] = byte(cfg.SnLeftHighThumb[0])
			frame[2] = byte(cfg.SnLeftHighThumb[1])
		}
	}

	// 设置其他手指的按压力度（跳过已处理的特殊拇指）
	for _, fingerName := range pressedFingers {
		if fingerName == "Thumb1" || fingerName == "Thumb2" {
			continue
		}
		fb.setFingerPressure(frame, fingerName, pressProfile)
	}
}

// BuildSaxophoneFrame 构建萨克斯指法帧
func (fb *FingeringBuilder) buildSaxophoneFrame(frame []byte, pressedFingers []string, pressProfile []int) {
	for _, fingerName := range pressedFingers {
		fb.setFingerPressure(frame, fingerName, pressProfile)
	}
}

// GetSuonaThumbType 获取唢呐拇指类型
func (fb *FingeringBuilder) getSuonaThumbType(pressedFingers []string) string {
	for _, finger := range pressedFingers {
		if finger == "Thumb1" || finger == "Thumb2" {
			return finger
		}
	}
	return ""
}

// SetFingerPressure 设置手指按压力度
func (fb *FingeringBuilder) setFingerPressure(frame []byte, fingerName string, pressProfile []int) {
	index := fb.getFingerIndex(fingerName)
	if index >= 0 && index < len(pressProfile) {
		frame[index+1] = byte(pressProfile[index])
	}
}

// GetFingerIndex 获取手指名称对应的数组索引
func (fb *FingeringBuilder) getFingerIndex(fingerName string) int {
	// 直接匹配
	if index, exists := fingerIndex[fingerName]; exists {
		return index
	}

	// 标准化名称后匹配
	normalized := strings.ToLower(strings.TrimSpace(fingerName))
	standardMappings := map[string]int{
		"thumb":          0,
		"thumb rotation": 1,
		"thumbrotation":  1,
		"index":          2,
		"middle":         3,
		"ring":           4,
		"little":         5,
		"pinky":          5,
		"thumb1":         0,
		"thumb2":         1,
	}

	if index, exists := standardMappings[normalized]; exists {
		return index
	}

	return -1 // 未识别的手指名称
}

// BuildReleaseFrame 构建释放数据帧（用于预备手势）
func (fb *FingeringBuilder) BuildReleaseFrame(releaseProfile []int) []byte {
	frame := make([]byte, 7)
	frame[0] = OpCode

	// 设置释放力度
	for i := 0; i < 6; i++ {
		if i < len(releaseProfile) {
			frame[i+1] = byte(releaseProfile[i])
		} else {
			frame[i+1] = 255
		}
	}

	return frame
}

// GetCurrentThumbState 获取当前音符的拇指状态
func (fb *FingeringBuilder) GetCurrentThumbState(leftFingers []string) string {
	for _, finger := range leftFingers {
		switch finger {
		case "Thumb1":
			return "Thumb1"
		case "Thumb2":
			return "Thumb2"
		default:
			return ""
		}
	}
	return "" // 没有高音拇指
}

// NeedsSmoothThumbTransition 检查是否需要唢呐拇指平滑切换
func (fb *FingeringBuilder) NeedsSmoothThumbTransition(lastState, currentState string) bool {
	// 只有在Thumb1和Thumb2之间切换时才需要平滑过渡
	return (lastState == "Thumb1" && currentState == "Thumb2") ||
		(lastState == "Thumb2" && currentState == "Thumb1")
}
