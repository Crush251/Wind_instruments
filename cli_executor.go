package main

import (
	"flag"
	"fmt"
	"os"
	"time"
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

// RunDirectPlayback 直接演奏模式
func (cli *CLIExecutor) RunDirectPlayback(inputFile, instrument, configFile string, bpmOverride float64, dryRun bool) {
	fmt.Printf("🎵 开始演奏: %s (%s)\n", inputFile, getInstrumentName(instrument))

	// 检查文件是否存在
	if err := cli.fileReader.CheckFileExists(inputFile); err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		os.Exit(1)
	}

	// 创建演奏引擎
	engine, err := newDirectPerformanceEngine(inputFile, instrument, configFile, bpmOverride, dryRun, cli.fileReader)
	if err != nil {
		fmt.Printf("❌ 错误: 创建演奏引擎失败: %v\n", err)
		os.Exit(1)
	}

	// 解析时间轴
	events, err := engine.parseTimeline(engine.timeline)
	if err != nil {
		fmt.Printf("❌ 错误: 解析时间轴失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 演奏信息: %d个音符, BPM: %.1f, 预计时长: %.1f秒\n",
		len(events), engine.getBPM(), engine.getEstimatedDuration(events))

	// 执行预备手势
	if engine.cfg.Ready.Enabled {
		fmt.Println("🤲 执行预备手势...")
		readyController := NewReadyGestureController()
		readyController.ExecuteReadyGestureWithDelay(engine.cfg, instrument, engine.cfg.Ready.HoldMS)
	}

	// 开始演奏
	fmt.Println("🎶 开始演奏...")
	startTime := time.Now()

	err = engine.playSequence(events)

	duration := time.Since(startTime)

	// 演奏结束处理
	utils := NewUtils()
	utils.ControlAirPumpWithLock(engine.cfg, false)
	readyController := NewReadyGestureController()
	readyController.ExecuteReadyGesture(engine.cfg, instrument)

	if err != nil {
		fmt.Printf("❌ 演奏过程中出现错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 演奏完成! 实际用时: %.1f秒\n", duration.Seconds())
}

// PrintUsage 打印使用说明
func (cli *CLIExecutor) PrintUsage() {
	fmt.Println("🎵 萨克斯/唢呐演奏控制系统")
	fmt.Println("\n用法:")
	fmt.Println("  直接演奏模式:")
	fmt.Println("    go run main.go -in trsmusic/test.json -instrument sks")
	fmt.Println("    go run main.go -in trsmusic/molihua.json -instrument sn -bpm 120")
	fmt.Println("\n  Web服务模式:")
	fmt.Println("    go run main.go -web")
	fmt.Println("    go run main.go  (默认启动Web服务)")
	fmt.Println("\n参数说明:")
	flag.PrintDefaults()
	fmt.Println("\n示例:")
	fmt.Println("  # 萨克斯演奏茉莉花")
	fmt.Println("  go run main.go -in trsmusic/molihua.json -instrument sks")
	fmt.Println("  # 唢呐演奏，指定BPM")
	fmt.Println("  go run main.go -in trsmusic/test.json -instrument sn -bpm 100")
	fmt.Println("  # 调试模式")
	fmt.Println("  go run main.go -in trsmusic/test.json -dry")
}

// GetInstrumentName 获取乐器中文名称
func getInstrumentName(instrument string) string {
	if instrument == "sn" {
		return "唢呐"
	}
	return "萨克斯"
}
