package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

////////////////////////////////////////////////////////////////////////////////
// 主程序入口
////////////////////////////////////////////////////////////////////////////////

func main() {
	// 设置信号处理，确保程序退出时正确关闭气泵控制器
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 收到退出信号，正在关闭气泵控制器...")
		CloseGlobalPumpController()
		os.Exit(0)
	}()

	// 定义命令行参数
	var (
		inputFile     = flag.String("in", "", "输入音乐文件路径 (例: trsmusic/test.json)")
		instrument    = flag.String("instrument", "sks", "乐器类型: sks(萨克斯) 或 sn(唢呐)")
		configFile    = flag.String("config", "config.yaml", "配置文件路径")
		bpmOverride   = flag.Float64("bpm", 0, "覆盖BPM设置 (0表示使用配置文件或JSON文件中的值)")
		tonguingDelay = flag.Int("tongue", 30, "吐音延迟时间（毫秒）")
		help          = flag.Bool("help", false, "显示帮助信息")
		preprocess    = flag.Bool("preprocess", false, "预处理模式：生成执行序列文件")
		outputFile    = flag.String("out", "", "预处理输出文件路径 (例: trsmusic/test.exec.json)")
		execFile      = flag.String("exec", "", "执行预计算的序列文件 (例: exec/test.exec.json)")
		jsonFile      = flag.String("json", "", "执行预计算的序列文件 (例: exec/test.exec.json) [-json 等同于 -exec]")
	)

	flag.Parse()

	// 处理 -json 和 -exec 参数（-json 优先级更高）
	if *jsonFile != "" {
		*execFile = *jsonFile
	}

	if *help {
		cliExecutor := NewCLIExecutor()
		cliExecutor.PrintUsage()
		return
	}

	// 加载配置文件
	fileReader := NewFileReader()
	cfg := fileReader.LoadConfig(*configFile)
	// 初始化气泵控制器（串口）
	if cfg.Pump.PortName != "" {
		fmt.Printf("🔧 正在初始化气泵控制器（串口）...\n")
		if err := InitGlobalPumpController(cfg.Pump.PortName); err != nil {
			fmt.Printf("❌ 气泵控制器初始化失败: %v\n", err)
			//os.Exit(1)
		}
	} else {
		fmt.Println("❌ 错误: 配置文件中未指定气泵串口")
		os.Exit(1)
	}
	// === 预处理模式 ===
	if *preprocess {
		if *inputFile == "" {
			fmt.Println("❌ 错误: 预处理模式需要指定输入文件 (-in)")
			os.Exit(1)
		}

		// 加载指法映射
		fingeringMap := fileReader.LoadFingeringMapByInstrument(*instrument)

		// 获取BPM
		bpm := *bpmOverride
		if bpm <= 0 {
			bpm = cfg.BPM
			if bpm <= 0 {
				bpm = 60 // 默认BPM
			}
		}

		// 自动生成输出文件名（如果未指定）
		if *outputFile == "" {
			// 确保 exec 目录存在
			if err := os.MkdirAll("exec", 0755); err != nil {
				fmt.Printf("❌ 错误: 创建 exec 目录失败: %v\n", err)
				os.Exit(1)
			}

			// 从输入文件路径提取基础文件名（去掉路径和.json扩展名）
			baseFilename := filepath.Base(*inputFile)
			baseFilename = baseFilename[:len(baseFilename)-5] // 移除 .json

			// 生成格式：原文件名_乐器类型_BPM_吐音延迟.exec.json
			// 例如：青花瓷-葫芦丝-4min-108_sn_108_30.exec.json
			*outputFile = fmt.Sprintf("exec/%s_%s_%.0f_%d.exec.json",
				baseFilename, *instrument, bpm, *tonguingDelay)

			fmt.Printf("📝 自动生成输出文件名: %s\n", *outputFile)
		}

		// 创建预处理器
		preprocessor := NewSequencePreprocessor(cfg, fingeringMap, *instrument, bpm, *tonguingDelay)

		// 生成执行序列
		if err := preprocessor.GenerateExecutionSequence(*inputFile, *outputFile); err != nil {
			fmt.Printf("❌ 预处理失败: %v\n", err)
			os.Exit(1)
		}

		return
	}

	// === 执行预计算序列模式 ===
	if *execFile != "" {
		// 初始化气泵控制器（串口）
		if cfg.Pump.PortName != "" {
			fmt.Printf("🔧 正在初始化气泵控制器（串口）...\n")
			if err := InitGlobalPumpController(cfg.Pump.PortName); err != nil {
				fmt.Printf("❌ 气泵控制器初始化失败: %v\n", err)
				//os.Exit(1)
			}
		} else {
			fmt.Println("❌ 错误: 配置文件中未指定气泵串口")
			os.Exit(1)
		}

		// 创建执行引擎
		engine, err := NewExecutionEngine(*execFile, cfg)
		if err != nil {
			fmt.Printf("❌ 创建执行引擎失败: %v\n", err)
			os.Exit(1)
		}

		// 执行播放
		if err := engine.Play(); err != nil {
			fmt.Printf("❌ 播放失败: %v\n", err)
			os.Exit(1)
		}

		// 演奏结束后关闭气泵控制器
		CloseGlobalPumpController()
		return
	}

	// === 自动预处理+执行模式 ===
	// 如果指定了输入文件，自动进行预处理后执行
	if *inputFile != "" {
		fmt.Println("🔄 检测到输入文件，自动进入预处理+执行模式...")

		// 加载指法映射
		fingeringMap := fileReader.LoadFingeringMapByInstrument(*instrument)

		// 获取BPM
		bpm := *bpmOverride
		if bpm <= 0 {
			bpm = cfg.BPM
			if bpm <= 0 {
				bpm = 60 // 默认BPM
			}
		}

		// 生成临时执行文件
		if err := os.MkdirAll("exec", 0755); err != nil {
			fmt.Printf("❌ 错误: 创建 exec 目录失败: %v\n", err)
			os.Exit(1)
		}

		baseFilename := filepath.Base(*inputFile)
		baseFilename = baseFilename[:len(baseFilename)-5] // 移除 .json
		tempExecFile := fmt.Sprintf("exec/%s_%s_%.0f_%d.exec.json",
			baseFilename, *instrument, bpm, *tonguingDelay)

		fmt.Printf("📝 第1步: 预处理生成执行序列 -> %s\n", tempExecFile)

		// 步骤1: 预处理
		preprocessor := NewSequencePreprocessor(cfg, fingeringMap, *instrument, bpm, *tonguingDelay)
		if err := preprocessor.GenerateExecutionSequence(*inputFile, tempExecFile); err != nil {
			fmt.Printf("❌ 预处理失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ 预处理完成")
		fmt.Println("🎵 第2步: 开始执行演奏...")

		// 步骤2: 初始化气泵控制器
		if cfg.Pump.PortName != "" {
			fmt.Printf("🔧 正在初始化气泵控制器（串口）...\n")
			if err := InitGlobalPumpController(cfg.Pump.PortName); err != nil {
				fmt.Printf("❌ 气泵控制器初始化失败: %v\n", err)
			}
		} else {
			fmt.Println("❌ 错误: 配置文件中未指定气泵串口")
			os.Exit(1)
		}

		// 步骤3: 执行播放
		engine, err := NewExecutionEngine(tempExecFile, cfg)
		if err != nil {
			fmt.Printf("❌ 创建执行引擎失败: %v\n", err)
			os.Exit(1)
		}

		if err := engine.Play(); err != nil {
			fmt.Printf("❌ 播放失败: %v\n", err)
			os.Exit(1)
		}

		// 演奏结束后关闭气泵控制器
		CloseGlobalPumpController()
		fmt.Println("✅ 演奏完成")
		return
	}

	// === Web服务模式 ===
	// 否则启动Web服务
	webServer := NewWebServer()
	webServer.StartWebServer()
}
