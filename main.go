package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
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
		inputFile   = flag.String("in", "", "输入音乐文件路径 (例: trsmusic/test.json)")
		instrument  = flag.String("instrument", "sks", "乐器类型: sks(萨克斯) 或 sn(唢呐)")
		configFile  = flag.String("config", "config.yaml", "配置文件路径")
		bpmOverride = flag.Float64("bpm", 0, "覆盖BPM设置 (0表示使用配置文件或JSON文件中的值)")
		dryRun      = flag.Bool("dry", false, "调试模式，只打印不发送CAN指令")
		help        = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Parse()

	if *help {
		cliExecutor := NewCLIExecutor()
		cliExecutor.PrintUsage()
		return
	}

	// 加载配置文件
	fileReader := NewFileReader()
	cfg := fileReader.LoadConfig(*configFile)

	// 初始化气泵控制器（如果配置为使用串口）
	if cfg.Pump.UseSerial && cfg.Pump.PortName != "" {
		fmt.Printf("🔧 正在初始化气泵控制器...\n")
		if err := InitGlobalPumpController(cfg.Pump.PortName); err != nil {
			fmt.Printf("⚠️  气泵控制器初始化失败: %v\n", err)
			fmt.Println("   将使用CAN通信方式")
		}
	} else {
		fmt.Println("🔧 使用CAN通信方式控制气泵")
	}

	// 如果指定了输入文件，直接演奏模式
	if *inputFile != "" {
		cliExecutor := NewCLIExecutor()
		cliExecutor.RunDirectPlayback(*inputFile, *instrument, *configFile, *bpmOverride, *dryRun)
		// 演奏结束后关闭气泵控制器
		CloseGlobalPumpController()
		return
	}

	// 否则启动Web服务
	webServer := NewWebServer()
	webServer.StartWebServer()
}

////////////////////////////////////////////////////////////////////////////////
// 二、演奏核心逻辑（顺序执行模式）
////////////////////////////////////////////////////////////////////////////////

// 创建演奏引擎（带参数版本）
func newPerformanceEngineWithParams(fpath string, instrument string, bpmOverride float64, tonguingDelay int, fileReader *FileReader) (*PerformanceEngine, error) {
	cfg := fileReader.LoadConfig("config.yaml")
	timeline := fileReader.LoadTimeline(fpath)

	// 根据乐器类型加载指法映射
	fingeringMap := fileReader.LoadFingeringMapByInstrument(instrument)

	// 获取BPM（优先使用传入的BPM）
	bpm := bpmOverride
	if bpm <= 0 {
		bpm = cfg.BPM
		if bpm <= 0 {
			if bpmVal, exists := timeline.Meta["bpm"]; exists {
				utils := NewUtils()
				if bpmFloat, ok := utils.ConvertToFloat(bpmVal); ok && bpmFloat > 0 {
					bpm = bpmFloat
				}
			}
			if bpm <= 0 {
				bpm = 60
			}
		}
	}

	// 更新控制器状态
	playbackController.mutex.Lock()
	playbackController.config = cfg
	playbackController.timeline = timeline
	playbackController.fingeringMap = fingeringMap
	playbackController.instrument = instrument
	playbackController.mutex.Unlock()

	return &PerformanceEngine{
		cfg:            cfg,
		fingeringMap:   fingeringMap,
		instrument:     instrument,
		secondsPerBeat: 60.0 / bpm,
		timeline:       timeline,
		tonguingDelay:  tonguingDelay,
	}, nil
}

// 创建演奏引擎（兼容旧版本）
func newPerformanceEngine(fpath string, instrument string, fileReader *FileReader) (*PerformanceEngine, error) {
	return newPerformanceEngineWithParams(fpath, instrument, 0, 30, fileReader)
}

// 创建直接演奏模式的演奏引擎
func newDirectPerformanceEngine(fpath, instrument, configFile string, bpmOverride float64, dryRun bool, fileReader *FileReader) (*PerformanceEngine, error) {
	cfg := fileReader.LoadConfig(configFile)
	timeline := fileReader.LoadTimeline(fpath)

	// 应用命令行覆盖
	if dryRun {
		cfg.DryRun = true
	}

	// 根据乐器类型加载指法映射
	fingeringMap := fileReader.LoadFingeringMapByInstrument(instrument)

	// 获取BPM（优先级：命令行 > 配置文件 > JSON文件 > 默认值）
	bpm := cfg.BPM
	if bpmOverride > 0 {
		bpm = bpmOverride
	} else if bpm <= 0 {
		if bpmVal, exists := timeline.Meta["bpm"]; exists {
			utils := NewUtils()
			if bpmFloat, ok := utils.ConvertToFloat(bpmVal); ok && bpmFloat > 0 {
				bpm = bpmFloat
			}
		}
		if bpm <= 0 {
			bpm = 60
		}
	}

	return &PerformanceEngine{
		cfg:            cfg,
		fingeringMap:   fingeringMap,
		instrument:     instrument,
		secondsPerBeat: 60.0 / bpm,
		timeline:       timeline,
		tonguingDelay:  30, // 默认吐音延迟30ms
	}, nil
}

// 获取BPM
func (pe *PerformanceEngine) getBPM() float64 {
	return 60.0 / pe.secondsPerBeat
}

// 估算演奏时长
func (pe *PerformanceEngine) getEstimatedDuration(events []NoteEvent) float64 {
	totalBeats := 0.0
	for _, event := range events {
		totalBeats += event.Duration
	}
	return totalBeats * pe.secondsPerBeat
}

// 解析时间轴为音符事件
func (pe *PerformanceEngine) parseTimeline(timeline TimelineFile) ([]NoteEvent, error) {
	var events []NoteEvent
	utils := NewUtils()

	for i, item := range timeline.Timeline {
		if len(item) < 2 {
			return nil, fmt.Errorf("第%d个音符数据不完整", i+1)
		}

		note, ok := item[0].(string)
		if !ok {
			return nil, fmt.Errorf("第%d个音符名称无效", i+1)
		}

		duration, ok := utils.ConvertToFloat(item[1])
		if !ok || duration <= 0 {
			return nil, fmt.Errorf("第%d个音符持续时间无效", i+1)
		}

		events = append(events, NoteEvent{
			Note:     note,
			Duration: duration,
			Index:    i + 1,
		})
	}
	return events, nil
}

// 异步开始演奏（带参数版本）
func startPerformanceAsyncWithParams(fpath string, instrument string, bpmOverride float64, tonguingDelay int, fileReader *FileReader) error {
	engine, err := newPerformanceEngineWithParams(fpath, instrument, bpmOverride, tonguingDelay, fileReader)
	if err != nil {
		return err
	}

	events, err := engine.parseTimeline(playbackController.timeline)
	if err != nil {
		return err
	}

	// 初始化演奏状态
	playbackController.mutex.Lock()
	playbackController.isRunning = true
	playbackController.startTime = time.Now()
	playbackController.status = PlaybackStatus{
		IsPlaying:   true,
		IsPaused:    false,
		CurrentFile: filepath.Base(fpath),
		CurrentNote: 0,
		TotalNotes:  len(events),
		Progress:    0,
	}
	playbackController.mutex.Unlock()

	// 执行预备手势
	if engine.cfg.Ready.Enabled {
		readyController := NewReadyGestureController()
		readyController.ExecuteReadyGestureWithDelay(engine.cfg, instrument, engine.cfg.Ready.HoldMS)
	}

	// 开始演奏序列
	err = engine.playSequence(events)

	// 演奏结束处理（确保气泵已关闭）
	utils := NewUtils()
	utils.ControlAirPumpWithLock(engine.cfg, false)
	readyController := NewReadyGestureController()
	readyController.ExecuteReadyGesture(engine.cfg, instrument)

	playbackController.mutex.Lock()
	playbackController.isRunning = false
	playbackController.status.IsPlaying = false
	playbackController.status.IsPaused = false
	playbackController.status.Progress = 100
	playbackController.status.CurrentFile = ""
	playbackController.status.CurrentNote = 0
	playbackController.status.TotalNotes = 0
	playbackController.status.ElapsedTime = ""
	playbackController.mutex.Unlock()

	return err
}

// 异步开始演奏（兼容旧版本）
func startPerformanceAsync(fpath string, instrument string, fileReader *FileReader) error {
	return startPerformanceAsyncWithParams(fpath, instrument, 0, 30, fileReader)
}

// 执行演奏序列（优化的吐音逻辑 + 对象复用 + 异步CAN发送）
func (pe *PerformanceEngine) playSequence(events []NoteEvent) error {
	// 对象复用：在循环外创建，避免重复分配内存和GC压力
	utils := NewUtils()
	readyController := NewReadyGestureController()

	fingeringPreSwitched := false // 标记指法是否已预切换
	skipNextCompensation := false // 标记下一个音符是否需要跳过时间补偿（因为已经在上一个音符处理过了）
	nextCompensation := 0.0       // 下一个音符需要扣除的时间（毫秒）

	for i, event := range events {
		// 检查控制信号
		if pe.checkControlSignals() {
			return nil
		}

		// 更新进度
		pe.updateProgress(i+1, len(events))

		// ================= 1. 休止符处理 =================
		if event.Note == "NO" {
			// 空拍处理
			// 1. 异步关闭气泵（不阻塞主程序）
			utils.ControlAirPumpAsync(pe.cfg, false)

			// 2. 异步切换到松开手指的指法（预备手势）- 使用复用的对象
			go readyController.ExecuteReadyGesture(pe.cfg, pe.instrument)

			duration := time.Duration(pe.secondsPerBeat*event.Duration*1000) * time.Millisecond

			// 3. 检查下一个音符，在休止符结束前20%时预切换指法
			nextIndex := i + 1
			if nextIndex < len(events) && events[nextIndex].Note != "NO" {
				// 计算预切换时间点（休止符结束前20%）
				preSwitchTime := time.Duration(float64(duration) * 0.2)
				normalWaitTime := duration - preSwitchTime

				// 先等待80%的时间
				time.Sleep(normalWaitTime)

				// 预切换到下一个音符的指法
				if err := pe.switchFingeringAsync(events[nextIndex].Note); err == nil {
					// 移除打印以提升性能
					// fmt.Printf("🎵 空拍中预切换指法: %s\n", events[nextIndex].Note)
					fingeringPreSwitched = true // 标记已预切换
				}

				// 等待剩余20%的时间
				time.Sleep(preSwitchTime)
			} else {
				// 如果下一个也是空拍或已到结尾，正常等待
				time.Sleep(duration)
			}

			// 重置补偿标记
			skipNextCompensation = false
			nextCompensation = 0.0
			continue // 跳过后续处理
		}

		// ================= 2. 非休止符处理 =================

		// 切换指法（如果未预切换）- 使用异步发送以提升速度
		if !fingeringPreSwitched {
			if err := pe.switchFingeringAsync(event.Note); err != nil {
				continue // 跳过无效音符，继续演奏
			}
		} else {
			// 指法已预切换，重置标志
			fingeringPreSwitched = false
			// 移除打印以提升性能
			// fmt.Printf("🎵 使用预切换的指法: %s\n", event.Note)
		}

		// 计算基本持续时间
		baseDuration := time.Duration(pe.secondsPerBeat*event.Duration*1000) * time.Millisecond
		playDuration := baseDuration

		// 如果这个音符需要扣除上一次计算的补偿时间
		if skipNextCompensation && nextCompensation > 0 {
			playDuration = baseDuration - time.Duration(nextCompensation)*time.Millisecond
			if playDuration < 0 {
				playDuration = 0
			}
			skipNextCompensation = false
			nextCompensation = 0.0
		}

		// 检查下一个音符是否与当前音符相同
		nextIndex := i + 1
		nextIsSame := false
		if nextIndex < len(events) && events[nextIndex].Note == event.Note && events[nextIndex].Note != "NO" {
			nextIsSame = true
		}

		// ================= 2.1 当前音符与下一个音符相同的处理 =================
		if nextIsSame {
			// 计算时间补偿：把 tongue_ms 按比例分配给当前音和下一个音
			currentDuration := event.Duration
			nextDuration := events[nextIndex].Duration
			totalDuration := currentDuration + nextDuration

			// gL: 当前音符承担的吐音延迟补偿
			gL := float64(pe.tonguingDelay) * (currentDuration / totalDuration)
			// gR: 下一个音符承担的吐音延迟补偿
			gR := float64(pe.tonguingDelay) * (nextDuration / totalDuration)

			// 当前音符播放时间 = base - gL
			playDuration = baseDuration - time.Duration(gL)*time.Millisecond
			if playDuration < 0 {
				playDuration = 0
			}

			// 异步打开气泵（不阻塞主程序）
			utils.ControlAirPumpAsync(pe.cfg, true)

			// *** 主程序严格按BPM时间推进 ***
			// 播放当前音符（已扣除 gL）
			if playDuration > 0 {
				time.Sleep(playDuration)
			}

			// 异步关闭气泵（不阻塞主程序）
			utils.ControlAirPumpAsync(pe.cfg, false)

			// *** 关键：插入实际的吐音间隙（主程序时间控制） ***
			time.Sleep(time.Duration(pe.tonguingDelay) * time.Millisecond)

			// 标记下一个音符需要扣除 gR
			skipNextCompensation = true
			nextCompensation = gR

		} else {
			// ================= 2.2 当前音符与下一个音符不同的处理 =================
			// 异步打开气泵（不阻塞主程序）
			utils.ControlAirPumpAsync(pe.cfg, true)

			// *** 主程序严格按BPM时间推进 ***
			// 正常播放完整时长
			time.Sleep(playDuration)

			// 保持气泵开启状态（下一个音符不同，不需要吐音）
		}
	}

	// 演奏结束，确保气泵关闭
	utils.ControlAirPumpWithLock(pe.cfg, false)

	return nil
}

// 检查控制信号
func (pe *PerformanceEngine) checkControlSignals() bool {
	select {
	case <-playbackController.stopChan:
		playbackController.mutex.Lock()
		playbackController.isRunning = false
		playbackController.status.IsPlaying = false
		playbackController.mutex.Unlock()
		return true
	case <-playbackController.pauseChan:
		playbackController.mutex.Lock()
		playbackController.status.IsPaused = true
		playbackController.mutex.Unlock()

		<-playbackController.resumeChan

		playbackController.mutex.Lock()
		playbackController.status.IsPaused = false
		playbackController.mutex.Unlock()
	default:
	}
	return false
}

// 更新演奏进度
func (pe *PerformanceEngine) updateProgress(current, total int) {
	playbackController.mutex.Lock()
	playbackController.status.CurrentNote = current
	playbackController.status.Progress = float64(current) / float64(total) * 100
	playbackController.status.ElapsedTime = time.Since(playbackController.startTime).Round(time.Second).String()
	playbackController.mutex.Unlock()
}

// 切换指法（同步版本，支持拇指状态追踪）
func (pe *PerformanceEngine) switchFingering(note string) error {
	fingering, exists := pe.fingeringMap[note]
	if !exists {
		return fmt.Errorf("未找到音符 %s 的指法映射", note)
	}

	return pe.sendFingeringFrames(fingering)
}

// 切换指法（异步版本，极速模式，不等待CAN响应）
func (pe *PerformanceEngine) switchFingeringAsync(note string) error {
	fingering, exists := pe.fingeringMap[note]
	if !exists {
		return fmt.Errorf("未找到音符 %s 的指法映射", note)
	}

	return pe.sendFingeringFramesAsync(fingering)
}

// 发送指法数据帧（优化的并发版本，支持唢呐拇指平滑切换）
func (pe *PerformanceEngine) sendFingeringFrames(fingering FingeringEntry) error {
	// 创建指法构建器
	fingeringBuilder := NewFingeringBuilder()

	// 根据乐器类型选择配置
	var leftPress, leftRelease, rightPress, rightRelease []int

	if pe.instrument == "sn" {
		leftPress = pe.cfg.SnLeftPressProfile
		leftRelease = pe.cfg.SnLeftReleaseProfile
		rightPress = pe.cfg.SnRightPressProfile
		rightRelease = pe.cfg.SnRightReleaseProfile
	} else {
		leftPress = pe.cfg.SksLeftPressProfile
		leftRelease = pe.cfg.SksLeftReleaseProfile
		rightPress = pe.cfg.SksRightPressProfile
		rightRelease = pe.cfg.SksRightReleaseProfile
	}

	// 检查是否需要唢呐拇指平滑切换
	if pe.instrument == "sn" {
		currentThumbState := fingeringBuilder.GetCurrentThumbState(fingering.Left)
		if fingeringBuilder.NeedsSmoothThumbTransition(pe.lastThumbState, currentThumbState) {
			// 先发送释放指令确保拇指平滑运动
			if err := pe.sendSmoothThumbTransition(leftPress, leftRelease); err != nil {
				return fmt.Errorf("拇指平滑切换失败: %v", err)
			}
			// 短暂延迟让拇指完成释放动作
			//time.Sleep(20 * time.Millisecond)
		}
		// 更新拇指状态
		pe.lastThumbState = currentThumbState
	}

	// 构建数据帧
	leftFrame := fingeringBuilder.BuildFingerFrame(fingering.Left, leftPress, leftRelease, pe.cfg, pe.instrument)
	rightFrame := fingeringBuilder.BuildFingerFrame(fingering.Right, rightPress, rightRelease, pe.cfg, pe.instrument)

	// 并发发送
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		utils := NewUtils()
		leftID := utils.ParseCanID(pe.cfg.Hands.Left.ID)
		if err := utils.SendCanFrame(pe.cfg, pe.cfg.Hands.Left.Interface, leftID, leftFrame); err != nil {
			errChan <- fmt.Errorf("左手指令发送失败: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		utils := NewUtils()
		rightID := utils.ParseCanID(pe.cfg.Hands.Right.ID)
		if err := utils.SendCanFrame(pe.cfg, pe.cfg.Hands.Right.Interface, rightID, rightFrame); err != nil {
			errChan <- fmt.Errorf("右手指令发送失败: %v", err)
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// 发送指法数据帧（异步版本，极速模式，不等待CAN响应）
func (pe *PerformanceEngine) sendFingeringFramesAsync(fingering FingeringEntry) error {
	// 创建指法构建器
	fingeringBuilder := NewFingeringBuilder()

	// 根据乐器类型选择配置
	var leftPress, leftRelease, rightPress, rightRelease []int

	if pe.instrument == "sn" {
		leftPress = pe.cfg.SnLeftPressProfile
		leftRelease = pe.cfg.SnLeftReleaseProfile
		rightPress = pe.cfg.SnRightPressProfile
		rightRelease = pe.cfg.SnRightReleaseProfile
	} else {
		leftPress = pe.cfg.SksLeftPressProfile
		leftRelease = pe.cfg.SksLeftReleaseProfile
		rightPress = pe.cfg.SksRightPressProfile
		rightRelease = pe.cfg.SksRightReleaseProfile
	}

	// 检查是否需要唢呐拇指平滑切换
	if pe.instrument == "sn" {
		currentThumbState := fingeringBuilder.GetCurrentThumbState(fingering.Left)
		if fingeringBuilder.NeedsSmoothThumbTransition(pe.lastThumbState, currentThumbState) {
			// 先发送释放指令确保拇指平滑运动（异步）
			pe.sendSmoothThumbTransitionAsync(leftPress, leftRelease)
		}
		// 更新拇指状态
		pe.lastThumbState = currentThumbState
	}

	// 构建数据帧
	leftFrame := fingeringBuilder.BuildFingerFrame(fingering.Left, leftPress, leftRelease, pe.cfg, pe.instrument)
	rightFrame := fingeringBuilder.BuildFingerFrame(fingering.Right, rightPress, rightRelease, pe.cfg, pe.instrument)

	// 异步并发发送（不等待响应）
	utils := NewUtils()
	leftID := utils.ParseCanID(pe.cfg.Hands.Left.ID)
	rightID := utils.ParseCanID(pe.cfg.Hands.Right.ID)

	utils.SendCanFrameAsync(pe.cfg, pe.cfg.Hands.Left.Interface, leftID, leftFrame)
	utils.SendCanFrameAsync(pe.cfg, pe.cfg.Hands.Right.Interface, rightID, rightFrame)

	return nil
}

// 发送唢呐拇指平滑切换的释放指令（同步版本）
func (pe *PerformanceEngine) sendSmoothThumbTransition(leftPress, leftRelease []int) error {
	// 构建释放数据帧
	fingeringBuilder := NewFingeringBuilder()
	releaseFrame := fingeringBuilder.BuildReleaseFrame(leftRelease)

	// 发送释放指令
	utils := NewUtils()
	leftID := utils.ParseCanID(pe.cfg.Hands.Left.ID)
	return utils.SendCanFrame(pe.cfg, pe.cfg.Hands.Left.Interface, leftID, releaseFrame)
}

// 发送唢呐拇指平滑切换的释放指令（异步版本）
func (pe *PerformanceEngine) sendSmoothThumbTransitionAsync(leftPress, leftRelease []int) {
	// 构建释放数据帧
	fingeringBuilder := NewFingeringBuilder()
	releaseFrame := fingeringBuilder.BuildReleaseFrame(leftRelease)

	// 异步发送释放指令
	utils := NewUtils()
	leftID := utils.ParseCanID(pe.cfg.Hands.Left.ID)
	utils.SendCanFrameAsync(pe.cfg, pe.cfg.Hands.Left.Interface, leftID, releaseFrame)
}
