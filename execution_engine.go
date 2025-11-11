package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// 特殊错误：用户停止播放
var ErrUserStopped = errors.New("user stopped playback")

////////////////////////////////////////////////////////////////////////////////
// 执行引擎 - 播放预计算的执行序列
////////////////////////////////////////////////////////////////////////////////

// ExecutionEngine 执行引擎
type ExecutionEngine struct {
	sequence     *ExecutionSequence
	cfg          Config
	httpClient   *http.Client
	utils        *Utils
	restTimings  []RestTiming   // 休止符时间记录
	actualStart  time.Time      // 实际开始时间
	actualEnd    time.Time      // 实际结束时间
	canWaitGroup sync.WaitGroup // 用于等待异步 CAN 帧完成
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

// Play 执行播放（使用 context 控制）
func (ee *ExecutionEngine) Play(ctx context.Context) error {
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
		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			fmt.Println("⏹️  收到停止信号，等待异步操作完成...")

			// 等待所有异步 CAN 帧完成（最多等待 100ms）
			done := make(chan struct{})
			go func() {
				ee.canWaitGroup.Wait()
				close(done)
			}()

			select {
			case <-done:
				fmt.Println("✅ 所有异步操作已完成")
			case <-time.After(100 * time.Millisecond):
				fmt.Println("⚠️  等待超时，强制停止")
			}

			return ErrUserStopped
		default:
		}

		// 更新进度
		ee.updateProgress(i+1, len(ee.sequence.Events))

		// 计算需要等待的时间（相对于上一个事件）
		waitDuration := time.Duration(event.TimestampMS-lastTimestamp) * time.Millisecond

		// 使用带上下文的 sleep，可以被立即打断
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
			case <-ctx.Done():
				return ErrUserStopped
			}
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
	// 异步执行串口气泵控制
	if event.SerialCmd != "" {
		ee.sendSerialCmd(event.SerialCmd)
	}

	// 使用 WaitGroup 跟踪所有 CAN 帧
	for _, frame := range event.Frames {
		ee.canWaitGroup.Add(1)
		go func(f ExecCANFrame) {
			defer ee.canWaitGroup.Done()
			ee.sendSingleFrame(f)
		}(frame)
	}
}

// sendSingleFrame 发送单个CAN帧
func (ee *ExecutionEngine) sendSingleFrame(frame ExecCANFrame) {
	if ee.cfg.DryRun {
		return
	}

	// 根据逻辑标识（left/right）映射到实际CAN接口
	var canInterface string
	switch frame.Hand {
	case "left":
		canInterface = ee.cfg.Hands.Left.Interface
	case "right":
		canInterface = ee.cfg.Hands.Right.Interface
	default:
		fmt.Printf("⚠️  警告: 未知的手部标识: %s\n", frame.Hand)
		return
	}

	// 解析ID
	var id uint32
	fmt.Sscanf(frame.ID, "0x%X", &id)

	// 使用异步发送
	ee.utils.SendCanFrameAsync(ee.cfg, canInterface, id, frame.Data)
}

// sendSerialCmd 发送串口命令
func (ee *ExecutionEngine) sendSerialCmd(cmd string) {
	if globalPumpController == nil {
		return
	}
	fmt.Println("给气泵发送命令: ", cmd)

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
	// 创建新的播放上下文
	ctx := playbackController.StartPlayback(ee.cfg, ee.sequence.Meta.Instrument)

	// 初始化状态
	playbackController.mutex.Lock()
	playbackController.status = PlaybackStatus{
		IsPlaying:   true,
		CurrentFile: ee.sequence.Meta.SourceFile,
		CurrentNote: 0,
		TotalNotes:  ee.sequence.Meta.TotalEvents,
		Progress:    0,
	}
	playbackController.mutex.Unlock()

	// 异步播放
	go func() {
		// 统一的资源清理（无论正常还是停止）
		defer func() {
			ee.cleanup()
		}()

		// 执行播放
		err := ee.Play(ctx)

		// 更新最终状态
		ee.updateFinalStatus(err)
	}()

	return nil
}

// cleanup 统一的资源清理函数
func (ee *ExecutionEngine) cleanup() {
	fmt.Println("🧹 开始资源清理...")

	// 1. 等待所有异步 CAN 帧完成
	done := make(chan struct{})
	go func() {
		ee.canWaitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("✅ 所有 CAN 帧已发送完成")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("⚠️  等待 CAN 帧超时")
	}

	// 2. 关闭气泵
	if globalPumpController != nil {
		fmt.Println("🔴 关闭气泵...")
		GlobalPumpOffSync()
	}

	// 3. 执行预备手势（松开手指）
	if playbackController.config.Ready.Enabled {
		fmt.Println("🤲 执行预备手势...")
		readyController := NewReadyGestureController()
		readyController.ExecuteReadyGesture(playbackController.config, ee.sequence.Meta.Instrument)
	}

	// 4. 标记播放完成
	playbackController.MarkFinished()

	fmt.Println("✅ 资源清理完成")
}

// updateFinalStatus 更新最终状态
func (ee *ExecutionEngine) updateFinalStatus(err error) {
	actualDuration := ee.actualEnd.Sub(ee.actualStart).Seconds()
	theoreticalDuration := ee.sequence.Meta.TotalDurationMS / 1000.0

	// 统计显著空拍
	significantRests := []RestTimingResponse{}
	for _, rest := range ee.restTimings {
		if rest.IsSignificant {
			startOffset := rest.StartTime.Sub(ee.actualStart).Seconds()
			endOffset := rest.EndTime.Sub(ee.actualStart).Seconds() / 0.8
			significantRests = append(significantRests, RestTimingResponse{
				StartOffset: startOffset,
				EndOffset:   endOffset,
				Duration:    rest.Duration / 0.8,
				Beats:       rest.Beats / 0.8,
			})
		}
	}

	playbackController.mutex.Lock()
	playbackController.status.Progress = 100
	playbackController.status.TheoreticalDuration = theoreticalDuration
	playbackController.status.ActualDuration = actualDuration
	playbackController.status.SignificantRests = significantRests
	playbackController.mutex.Unlock()

	if err != nil {
		if errors.Is(err, ErrUserStopped) {
			fmt.Printf("⏹️  播放已被用户停止\n")
		} else {
			fmt.Printf("❌ 播放出错: %v\n", err)
		}
	} else {
		fmt.Printf("✅ 播放完成\n")
	}
}
