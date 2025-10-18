package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// 特殊错误：用户停止播放
var ErrUserStopped = errors.New("user stopped playback")

////////////////////////////////////////////////////////////////////////////////
// 执行引擎 - 播放预计算的执行序列
////////////////////////////////////////////////////////////////////////////////

// ExecutionEngine 执行引擎
type ExecutionEngine struct {
	sequence    *ExecutionSequence
	cfg         Config
	httpClient  *http.Client
	utils       *Utils
	restTimings []RestTiming // 休止符时间记录
	actualStart time.Time    // 实际开始时间
	actualEnd   time.Time    // 实际结束时间
}

// RestTiming 休止符时间记录
type RestTiming struct {
	StartTime     time.Time // 休止符开始时间
	EndTime       time.Time // 休止符结束时间
	Duration      float64   // 持续时长（秒）
	DurationMS    float64   // 持续时长（毫秒）
	Beats         float64   // 拍数
	IsSignificant bool      // 是否为显著空拍（≥4拍或≥1秒）
}

// NewExecutionEngine 创建新的执行引擎
func NewExecutionEngine(sequenceFile string, cfg Config) (*ExecutionEngine, error) {
	// 加载执行序列
	sequence, err := loadExecutionSequence(sequenceFile)
	if err != nil {
		return nil, fmt.Errorf("加载执行序列失败: %v", err)
	}

	return &ExecutionEngine{
		sequence:   sequence,
		cfg:        cfg,
		httpClient: InitGlobalHTTPClient(),
		utils:      NewUtils(),
	}, nil
}

// loadExecutionSequence 加载执行序列文件
func loadExecutionSequence(filepath string) (*ExecutionSequence, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	var sequence ExecutionSequence
	if err := json.Unmarshal(data, &sequence); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return &sequence, nil
}

// Play 执行播放（极简版本，主程序只负责时间控制）
func (ee *ExecutionEngine) Play() error {
	fmt.Printf("🎵 开始执行播放\n")
	fmt.Printf("   文件: %s\n", ee.sequence.Meta.SourceFile)
	fmt.Printf("   乐器: %s, BPM: %.1f\n", ee.sequence.Meta.Instrument, ee.sequence.Meta.BPM)
	fmt.Printf("   事件数: %d, 总时长: %.2fs\n",
		ee.sequence.Meta.TotalEvents,
		ee.sequence.Meta.TotalDurationMS/1000.0)

	startTime := time.Now()
	ee.actualStart = startTime
	lastTimestamp := 0.0

	// 计算每拍的毫秒数
	msPerBeat := (60.0 / ee.sequence.Meta.BPM) * 1000.0

	for i, event := range ee.sequence.Events {
		// 检查停止信号
		select {
		case <-playbackController.stopChan:
			fmt.Println("⏹️  收到停止信号，正在关闭气泵...")
			// 立即关闭气泵
			if globalPumpController != nil {
				GlobalPumpOff()
				fmt.Println("🔴 气泵已紧急关闭")
			}
			return ErrUserStopped
		default:
		}

		// 更新进度
		ee.updateProgress(i+1, len(ee.sequence.Events))

		// 计算需要等待的时间（相对于上一个事件）
		waitDuration := time.Duration(event.TimestampMS-lastTimestamp) * time.Millisecond

		// *** 主程序只负责精确时间控制 ***
		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}

		// *** 所有I/O操作异步执行（不阻塞主程序） ***
		ee.sendFramesAsync(event)

		// 记录休止符时间
		if event.Note == "REST" {
			// 记录休止符开始时间
			restStart := time.Now()
			beats := event.DurationMS / msPerBeat
			ee.restTimings = append(ee.restTimings, RestTiming{
				StartTime:  restStart,
				DurationMS: event.DurationMS,
				Beats:      beats,
			})
		} else if len(ee.restTimings) > 0 && ee.restTimings[len(ee.restTimings)-1].EndTime.IsZero() {
			// 记录休止符结束时间
			idx := len(ee.restTimings) - 1
			ee.restTimings[idx].EndTime = time.Now()
			ee.restTimings[idx].Duration = ee.restTimings[idx].EndTime.Sub(ee.restTimings[idx].StartTime).Seconds()

			// 判断是否为显著空拍（≥4拍 或 ≥1秒）
			if ee.restTimings[idx].Beats >= 4.0 || ee.restTimings[idx].Duration >= 1.0 {
				ee.restTimings[idx].IsSignificant = true
			}
		}

		lastTimestamp = event.TimestampMS
	}

	ee.actualEnd = time.Now()
	elapsed := time.Since(startTime)

	// 统计显著空拍
	significantRests := []RestTiming{}
	for _, rest := range ee.restTimings {
		if rest.IsSignificant {
			significantRests = append(significantRests, rest)
		}
	}

	fmt.Printf("✅ 播放完成\n")
	fmt.Printf("   理论时长: %.2fs\n", ee.sequence.Meta.TotalDurationMS/1000.0)
	fmt.Printf("   实际时长: %.2fs\n", elapsed.Seconds())
	fmt.Printf("   时间误差: %.3fs (%.2f%%)\n",
		elapsed.Seconds()-ee.sequence.Meta.TotalDurationMS/1000.0,
		(elapsed.Seconds()-ee.sequence.Meta.TotalDurationMS/1000.0)/(ee.sequence.Meta.TotalDurationMS/1000.0)*100)
	fmt.Printf("   休止符次数: %d (显著空拍: %d)\n", len(ee.restTimings), len(significantRests))

	// 打印显著空拍详情
	if len(significantRests) > 0 {
		fmt.Printf("\n📊 显著空拍详情 (≥4拍或≥1秒):\n")
		for i, rest := range significantRests {
			startOffset := rest.StartTime.Sub(startTime).Seconds()
			endOffset := rest.EndTime.Sub(startTime).Seconds()
			fmt.Printf("   空拍%d: 起始%.2fs, 结束%.2fs, 持续%.2fs (%.1f拍)\n",
				i+1, startOffset, endOffset, rest.Duration, rest.Beats)
		}
	}

	return nil
}

// sendFramesAsync 异步发送所有CAN帧和串口命令
func (ee *ExecutionEngine) sendFramesAsync(event ExecutionEvent) {
	// 异步发送所有CAN帧（指法）
	for _, frame := range event.Frames {
		go ee.sendSingleFrame(frame)
	}

	// 异步执行串口气泵控制
	if event.SerialCmd != "" {
		go ee.sendSerialCmd(event.SerialCmd)
	}
}

// sendSingleFrame 发送单个CAN帧
func (ee *ExecutionEngine) sendSingleFrame(frame ExecCANFrame) {
	if ee.cfg.DryRun {
		return
	}

	// 解析ID
	var id uint32
	fmt.Sscanf(frame.ID, "0x%X", &id)

	// 使用异步发送
	ee.utils.SendCanFrameAsync(ee.cfg, frame.Interface, id, frame.Data)
}

// sendSerialCmd 发送串口命令
func (ee *ExecutionEngine) sendSerialCmd(cmd string) {
	if globalPumpController == nil {
		return
	}

	switch cmd {
	case "on":
		GlobalPumpOn()
	case "off":
		GlobalPumpOff()
	}
}

// updateProgress 更新播放进度
func (ee *ExecutionEngine) updateProgress(current, total int) {
	playbackController.mutex.Lock()
	playbackController.status.CurrentNote = current
	playbackController.status.Progress = float64(current) / float64(total) * 100
	playbackController.status.ElapsedTime = time.Since(playbackController.startTime).Round(time.Second).String()
	playbackController.mutex.Unlock()
}

// PlayAsync 异步执行播放（用于Web API）
func (ee *ExecutionEngine) PlayAsync() error {
	// 初始化演奏状态
	playbackController.mutex.Lock()
	playbackController.isRunning = true
	playbackController.startTime = time.Now()
	playbackController.instrument = ee.sequence.Meta.Instrument // 设置乐器类型
	playbackController.config = ee.cfg                          // 设置配置
	playbackController.status = PlaybackStatus{
		IsPlaying:   true,
		CurrentFile: ee.sequence.Meta.SourceFile,
		CurrentNote: 0,
		TotalNotes:  ee.sequence.Meta.TotalEvents,
		Progress:    0,
	}
	playbackController.mutex.Unlock()

	// 开始播放
	go func() {
		defer func() {
			// 确保播放结束时发送完成信号
			select {
			case playbackController.doneChan <- true:
				fmt.Println("📢 播放goroutine: 已发送完成信号")
			default:
				fmt.Println("⚠️  播放goroutine: 完成信号通道已满")
			}
		}()

		err := ee.Play()

		// 播放结束处理 - 确保气泵关闭
		if globalPumpController != nil {
			GlobalPumpOff()
		}

		// 执行预备手势（松开手指）
		if playbackController.config.Ready.Enabled {
			readyController := NewReadyGestureController()
			readyController.ExecuteReadyGesture(playbackController.config, ee.sequence.Meta.Instrument)
		}

		// 计算实际播放时长
		actualDuration := ee.actualEnd.Sub(ee.actualStart).Seconds()
		theoreticalDuration := ee.sequence.Meta.TotalDurationMS / 1000.0

		// 统计显著空拍
		significantRests := []RestTimingResponse{}
		for _, rest := range ee.restTimings {
			if rest.IsSignificant {
				startOffset := rest.StartTime.Sub(ee.actualStart).Seconds()
				// 修正结束时间：因为记录的是预切换时刻（80%处），需要除以0.8得到实际结束时间
				endOffset := rest.EndTime.Sub(ee.actualStart).Seconds() / 0.8
				significantRests = append(significantRests, RestTimingResponse{
					StartOffset: startOffset,
					EndOffset:   endOffset,
					Duration:    rest.Duration / 0.8, //修正时长：因为记录的是预切换时刻（80%处），需要除以0.8得到实际时长
					Beats:       rest.Beats / 0.8,    //修正拍数：因为记录的是预切换时刻（80%处），需要除以0.8得到实际拍数
				})
			}
		}

		// 更新播放状态（包含空拍信息）
		playbackController.mutex.Lock()
		playbackController.isRunning = false
		playbackController.status.IsPlaying = false
		playbackController.status.Progress = 100
		playbackController.status.TheoreticalDuration = theoreticalDuration
		playbackController.status.ActualDuration = actualDuration
		playbackController.status.SignificantRests = significantRests
		// 保留 CurrentFile、CurrentNote、TotalNotes 以便前端显示
		playbackController.mutex.Unlock()

		if err != nil {
			if errors.Is(err, ErrUserStopped) {
				fmt.Printf("⏹️  播放已被用户停止\n")
			} else {
				fmt.Printf("❌ 播放出错: %v\n", err)
			}
		} else {
			fmt.Printf("✅ 播放完成，气泵已关闭\n")
		}
	}()

	return nil
}
