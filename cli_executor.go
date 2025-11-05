package main

import (
	"flag"
	"fmt"
)

////////////////////////////////////////////////////////////////////////////////
// 命令行执行模块
////////////////////////////////////////////////////////////////////////////////

// CLIExecutor 命令行执行器
type CLIExecutor struct {
	fileReader *FileReader
}

// NewCLIExecutor 创建新的命令行执行器
func NewCLIExecutor() *CLIExecutor {
	return &CLIExecutor{
		fileReader: NewFileReader(),
	}
}

// PrintUsage 打印使用说明
func (cli *CLIExecutor) PrintUsage() {
	fmt.Println("🎵 萨克斯/唢呐演奏控制系统")
	fmt.Println("\n用法:")
	fmt.Println("  1. 执行预计算序列（最快，推荐）:")
	fmt.Println("    ./newsksgo -json exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json")
	fmt.Println("    ./newsksgo -exec exec/茉莉花_sks_120_30.exec.json")
	fmt.Println("\n  2. 预处理模式（生成exec文件）:")
	fmt.Println("    ./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 108 -tongue 30")
	fmt.Println("    → 自动生成: exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json")
	fmt.Println("\n  3. 自动预处理+执行模式（一步到位）:")
	fmt.Println("    ./newsksgo -in trsmusic/test.json -instrument sks -bpm 120 -tongue 30")
	fmt.Println("    → 自动预处理并立即演奏")
	fmt.Println("\n  4. Web服务模式:")
	fmt.Println("    ./newsksgo")
	fmt.Println("    ./newsksgo -config config.yaml")
	fmt.Println("\n参数说明:")
	flag.PrintDefaults()
	fmt.Println("\n完整示例:")
	fmt.Println("  # 预处理：生成exec文件（自动命名）")
	fmt.Println("  ./newsksgo -preprocess -in trsmusic/青花瓷-葫芦丝-4min-108.json -instrument sn -bpm 108 -tongue 30")
	fmt.Println("  → 生成文件: exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json")
	fmt.Println("")
	fmt.Println("  # 执行预计算的音乐序列（最快）")
	fmt.Println("  ./newsksgo -json exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json")
	fmt.Println("")
	fmt.Println("  # 手动指定输出文件名")
	fmt.Println("  ./newsksgo -preprocess -in trsmusic/茉莉花.json -instrument sks -bpm 120 -out exec/my_custom_name.exec.json")
	fmt.Println("")
	fmt.Println("  # 启动Web服务（默认监听8088端口）")
	fmt.Println("  ./newsksgo")
	fmt.Println("\n文件命名规则:")
	fmt.Println("  格式: exec/{原文件名}_{乐器类型}_{BPM}_{吐音延迟}.exec.json")
	fmt.Println("  示例: exec/青花瓷-葫芦丝-4min-108_sn_108_30.exec.json")
	fmt.Println("        └─ 青花瓷-葫芦丝-4min-108: 原音乐文件名")
	fmt.Println("        └─ sn: 乐器类型 (sn=唢呐, sks=萨克斯)")
	fmt.Println("        └─ 108: BPM (每分钟节拍数)")
	fmt.Println("        └─ 30: 吐音延迟 (毫秒)")
}
