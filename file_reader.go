package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

////////////////////////////////////////////////////////////////////////////////
// 文件读取器模块
////////////////////////////////////////////////////////////////////////////////

// FileReader 文件读取器接口
type FileReader struct{}

// NewFileReader 创建新的文件读取器
func NewFileReader() *FileReader {
	return &FileReader{}
}

// LoadConfig 加载主配置文件
func (fr *FileReader) LoadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ 错误: 无法读取配置文件 %s: %v\n", path, err)
		os.Exit(1)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("❌ 错误: 配置文件格式错误 %s: %v\n", path, err)
		os.Exit(1)
	}

	// 设置默认值（注意：BPM在main函数中处理，支持从JSON文件读取）
	if cfg.CanBridgeURL == "" {
		cfg.CanBridgeURL = "http://localhost:5260"
	}
	if cfg.Hands.Left.Interface == "" {
		cfg.Hands.Left.Interface = "can0"
	}
	if cfg.Hands.Right.Interface == "" {
		cfg.Hands.Right.Interface = "can1"
	}

	return cfg
}

// LoadTimeline 加载时间轴文件
func (fr *FileReader) LoadTimeline(path string) TimelineFile {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ 错误: 无法读取时间轴文件 %s: %v\n", path, err)
		os.Exit(1)
	}

	var timeline TimelineFile
	if err := json.Unmarshal(data, &timeline); err != nil {
		fmt.Printf("❌ 错误: 时间轴文件格式错误 %s: %v\n", path, err)
		os.Exit(1)
	}

	if len(timeline.Timeline) == 0 {
		fmt.Printf("❌ 错误: 时间轴文件为空 %s\n", path)
		os.Exit(1)
	}

	return timeline
}

// LoadFingeringMap 加载指法映射文件
func (fr *FileReader) LoadFingeringMap(path string) map[string]FingeringEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ 错误: 无法读取指法映射文件 %s: %v\n", path, err)
		os.Exit(1)
	}

	var cfg FingeringConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("❌ 错误: 指法映射文件格式错误 %s: %v\n", path, err)
		os.Exit(1)
	}

	// 转换为map便于查找
	fingeringMap := make(map[string]FingeringEntry)
	for _, entry := range cfg.FingeringMap {
		fingeringMap[entry.Note] = entry
	}

	return fingeringMap
}

// LoadFingeringMapByInstrument 根据乐器类型加载指法映射
func (fr *FileReader) LoadFingeringMapByInstrument(instrument string) map[string]FingeringEntry {
	var fingeringPath string
	if instrument == "sn" {
		fingeringPath = "config/snFinger.yaml"
	} else {
		fingeringPath = "config/sksFinger.yaml"
	}
	return fr.LoadFingeringMap(fingeringPath)
}

// CheckFileExists 检查文件是否存在
func (fr *FileReader) CheckFileExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", path)
	}
	return nil
}





















