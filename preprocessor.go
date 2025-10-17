package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// 执行序列预处理器
////////////////////////////////////////////////////////////////////////////////

// SequencePreprocessor 序列预处理器
type SequencePreprocessor struct {
	cfg            Config
	fingeringMap   map[string]FingeringEntry
	instrument     string
	bpm            float64
	tonguingDelay  int
	secondsPerBeat float64
}

// NewSequencePreprocessor 创建新的序列预处理器
func NewSequencePreprocessor(cfg Config, fingeringMap map[string]FingeringEntry, instrument string, bpm float64, tonguingDelay int) *SequencePreprocessor {
	return &SequencePreprocessor{
		cfg:            cfg,
		fingeringMap:   fingeringMap,
		instrument:     instrument,
		bpm:            bpm,
		tonguingDelay:  tonguingDelay,
		secondsPerBeat: 60.0 / bpm,
	}
}

// GenerateExecutionSequence 生成执行序列文件
func (sp *SequencePreprocessor) GenerateExecutionSequence(musicFile string, outputFile string) error {
	fmt.Printf("🔄 开始预处理: %s\n", musicFile)
	fmt.Printf("   乐器: %s, BPM: %.1f, 吐音延迟: %dms\n", sp.instrument, sp.bpm, sp.tonguingDelay)

	// 1. 加载时间轴文件
	fileReader := NewFileReader()
	timeline := fileReader.LoadTimeline(musicFile)

	// 2. 解析为音符事件
	events, err := sp.parseTimeline(timeline)
	if err != nil {
		return fmt.Errorf("解析时间轴失败: %v", err)
	}

	fmt.Printf("   音符总数: %d\n", len(events))

	// 3. 生成执行序列
	execSequence, err := sp.generateSequence(events, musicFile)
	if err != nil {
		return fmt.Errorf("生成执行序列失败: %v", err)
	}

	fmt.Printf("   执行事件数: %d\n", len(execSequence.Events))
	fmt.Printf("   总时长: %.2f秒\n", execSequence.Meta.TotalDurationMS/1000.0)

	// 4. 保存为JSON文件
	if err := sp.saveSequence(execSequence, outputFile); err != nil {
		return fmt.Errorf("保存执行序列失败: %v", err)
	}

	fmt.Printf("✅ 预处理完成: %s\n", outputFile)
	return nil
}

// parseTimeline 解析时间轴为音符事件
func (sp *SequencePreprocessor) parseTimeline(timeline TimelineFile) ([]NoteEvent, error) {
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

// generateSequence 生成执行序列
func (sp *SequencePreprocessor) generateSequence(events []NoteEvent, sourceFile string) (*ExecutionSequence, error) {
	sequence := &ExecutionSequence{
		Meta: SequenceMeta{
			SourceFile:    filepath.Base(sourceFile),
			Instrument:    sp.instrument,
			BPM:           sp.bpm,
			TonguingDelay: sp.tonguingDelay,
			GeneratedAt:   time.Now(),
			Version:       "1.0",
		},
		Events: []ExecutionEvent{},
	}

	currentTimeMS := 0.0
	rightCompensation := 0.0 // 从上一个音符继承的右侧补偿

	for i, event := range events {
		baseDurationMS := sp.secondsPerBeat * event.Duration * 1000.0

		// 根据音符类型生成不同的执行事件
		if event.Note == "NO" {
			// 空拍处理
			execEvents, err := sp.generateRestEvents(currentTimeMS, baseDurationMS, i, events)
			if err != nil {
				return nil, err
			}
			sequence.Events = append(sequence.Events, execEvents...)
			currentTimeMS += baseDurationMS
			rightCompensation = 0.0 // 空拍后重置补偿

		} else {
			// 检查上一个和下一个音符是否与当前音符相同
			prevIndex := i - 1
			nextIndex := i + 1

			prevIsSame := false
			if prevIndex >= 0 && events[prevIndex].Note == event.Note && events[prevIndex].Note != "NO" {
				prevIsSame = true
			}

			nextIsSame := false
			if nextIndex < len(events) && events[nextIndex].Note == event.Note && events[nextIndex].Note != "NO" {
				nextIsSame = true
			}

			// 计算当前音符的补偿
			leftCompensation := rightCompensation // 继承上一个音符的右侧补偿
			rightCompensation = 0.0               // 重置，如果需要会重新计算

			if nextIsSame {
				// 下一个音符相同，需要计算补偿
				currentDuration := event.Duration
				nextDuration := events[nextIndex].Duration
				totalDuration := currentDuration + nextDuration

				// 按比例分配吐音延迟
				gL := float64(sp.tonguingDelay) * (currentDuration / totalDuration)
				gR := float64(sp.tonguingDelay) * (nextDuration / totalDuration)

				// 如果已经有左侧补偿（中间音符），则累加
				leftCompensation += gL
				rightCompensation = gR
			}

			// 计算实际播放时长
			playDurationMS := baseDurationMS - leftCompensation
			if playDurationMS < 0 {
				playDurationMS = 0
			}

			if prevIsSame {
				// 与上一个音符相同（吐音续接）
				execEvents, err := sp.generateTonguingContinuation(currentTimeMS, playDurationMS, event, nextIsSame)
				if err != nil {
					return nil, err
				}
				sequence.Events = append(sequence.Events, execEvents...)
				// 只有当下一个还是相同音符时，才加上吐音延迟
				if nextIsSame {
					currentTimeMS += playDurationMS + float64(sp.tonguingDelay)
				} else {
					currentTimeMS += playDurationMS
				}

			} else {
				// 新音符或首次出现
				if nextIsSame {
					// 下一个相同，生成吐音开始
					execEvents, err := sp.generateTonguingStart(currentTimeMS, playDurationMS, event, nextIsSame)
					if err != nil {
						return nil, err
					}
					sequence.Events = append(sequence.Events, execEvents...)
					currentTimeMS += playDurationMS + float64(sp.tonguingDelay)
				} else {
					// 普通音符
					execEvent, err := sp.generateNormalEvent(currentTimeMS, playDurationMS, event)
					if err != nil {
						return nil, err
					}
					sequence.Events = append(sequence.Events, execEvent)
					currentTimeMS += playDurationMS
				}
			}
		}
	}

	// 演奏结束：关闭气泵和松开手指
	sequence.Events = append(sequence.Events, sp.generateEndEvent(currentTimeMS))

	// 更新元数据
	sequence.Meta.TotalDurationMS = currentTimeMS
	sequence.Meta.TotalEvents = len(sequence.Events)

	return sequence, nil
}

// generateNormalEvent 生成普通音符事件
func (sp *SequencePreprocessor) generateNormalEvent(timestampMS, durationMS float64, event NoteEvent) (ExecutionEvent, error) {
	frames, err := sp.buildFingeringFrames(event.Note)
	if err != nil {
		return ExecutionEvent{}, err
	}

	// 气泵通过串口控制
	return ExecutionEvent{
		TimestampMS: timestampMS,
		DurationMS:  durationMS,
		Note:        event.Note,
		Frames:      frames,
		SerialCmd:   "on",
	}, nil
}

// generateTonguingStart 生成吐音开始事件（第一个相同音符）
// 参数 nextIsSame: 下一个音符是否还是相同音符，决定是否添加吐音间隙
func (sp *SequencePreprocessor) generateTonguingStart(timestampMS, playDurationMS float64, event NoteEvent, nextIsSame bool) ([]ExecutionEvent, error) {
	events := []ExecutionEvent{}

	// 生成指法帧（第一个音符需要切换指法）
	frames, err := sp.buildFingeringFrames(event.Note)
	if err != nil {
		return nil, err
	}

	// 事件1: 切换指法 + 开启气泵
	events = append(events, ExecutionEvent{
		TimestampMS: timestampMS,
		DurationMS:  playDurationMS,
		Note:        event.Note,
		Frames:      frames, // ✅ 包含指法帧
		SerialCmd:   "on",
	})

	// 事件2: 关闭气泵（吐音间隙）- 仅当下一个音符还是相同时才添加
	if nextIsSame {
		events = append(events, ExecutionEvent{
			TimestampMS: timestampMS + playDurationMS,
			DurationMS:  float64(sp.tonguingDelay),
			Note:        "TONGUE",
			Frames:      []ExecCANFrame{},
			SerialCmd:   "off",
		})
	}

	return events, nil
}

// generateTonguingContinuation 生成吐音续接事件（后续相同音符）
// 参数 nextIsSame: 下一个音符是否还是相同音符，决定是否添加吐音间隙
func (sp *SequencePreprocessor) generateTonguingContinuation(timestampMS, playDurationMS float64, event NoteEvent, nextIsSame bool) ([]ExecutionEvent, error) {
	events := []ExecutionEvent{}

	// 事件1: 开启气泵（指法不变，无需CAN帧）
	events = append(events, ExecutionEvent{
		TimestampMS: timestampMS,
		DurationMS:  playDurationMS,
		Note:        event.Note,
		Frames:      []ExecCANFrame{}, // 无CAN帧，指法已设置
		SerialCmd:   "on",
	})

	// 事件2: 关闭气泵（吐音间隙）- 仅当下一个音符还是相同时才添加
	if nextIsSame {
		events = append(events, ExecutionEvent{
			TimestampMS: timestampMS + playDurationMS,
			DurationMS:  float64(sp.tonguingDelay),
			Note:        "TONGUE",
			Frames:      []ExecCANFrame{},
			SerialCmd:   "off",
		})
	}

	return events, nil
}

// generateRestEvents 生成空拍事件
func (sp *SequencePreprocessor) generateRestEvents(timestampMS, durationMS float64, currentIndex int, allEvents []NoteEvent) ([]ExecutionEvent, error) {
	events := []ExecutionEvent{}

	// 事件1: 关闭气泵 + 释放手指
	releaseFrames := sp.buildReleaseFrames()

	events = append(events, ExecutionEvent{
		TimestampMS: timestampMS,
		DurationMS:  durationMS * 0.8, // 80%时间
		Note:        "REST",
		Frames:      releaseFrames,
		SerialCmd:   "off",
	})

	// 检查是否需要预切换下一个音符的指法
	nextIndex := currentIndex + 1
	if nextIndex < len(allEvents) && allEvents[nextIndex].Note != "NO" {
		// 事件2: 在空拍结束前20%时预切换指法
		nextFingeringFrames, err := sp.buildFingeringFrames(allEvents[nextIndex].Note)
		if err == nil {
			events = append(events, ExecutionEvent{
				TimestampMS: timestampMS + durationMS*0.8,
				DurationMS:  durationMS * 0.2, // 剩余20%时间
				Note:        fmt.Sprintf("PRE_%s", allEvents[nextIndex].Note),
				Frames:      nextFingeringFrames,
				SerialCmd:   "",
			})
		}
	}

	return events, nil
}

// generateEndEvent 生成演奏结束事件
func (sp *SequencePreprocessor) generateEndEvent(timestampMS float64) ExecutionEvent {
	releaseFrames := sp.buildReleaseFrames()

	return ExecutionEvent{
		TimestampMS: timestampMS,
		DurationMS:  0,
		Note:        "END",
		Frames:      releaseFrames,
		SerialCmd:   "off",
	}
}

// buildFingeringFrames 构建指法CAN帧
func (sp *SequencePreprocessor) buildFingeringFrames(note string) ([]ExecCANFrame, error) {
	fingering, exists := sp.fingeringMap[note]
	if !exists {
		return nil, fmt.Errorf("未找到音符 %s 的指法映射", note)
	}

	fingeringBuilder := NewFingeringBuilder()
	utils := NewUtils()

	// 根据乐器类型选择配置
	var leftPress, leftRelease, rightPress, rightRelease []int
	if sp.instrument == "sn" {
		leftPress = sp.cfg.SnLeftPressProfile
		leftRelease = sp.cfg.SnLeftReleaseProfile
		rightPress = sp.cfg.SnRightPressProfile
		rightRelease = sp.cfg.SnRightReleaseProfile
	} else {
		leftPress = sp.cfg.SksLeftPressProfile
		leftRelease = sp.cfg.SksLeftReleaseProfile
		rightPress = sp.cfg.SksRightPressProfile
		rightRelease = sp.cfg.SksRightReleaseProfile
	}

	// 构建数据帧
	leftFrame := fingeringBuilder.BuildFingerFrame(fingering.Left, leftPress, leftRelease, sp.cfg, sp.instrument)
	rightFrame := fingeringBuilder.BuildFingerFrame(fingering.Right, rightPress, rightRelease, sp.cfg, sp.instrument)

	// 转换为执行帧
	leftID := utils.ParseCanID(sp.cfg.Hands.Left.ID)
	rightID := utils.ParseCanID(sp.cfg.Hands.Right.ID)

	return []ExecCANFrame{
		{
			Interface: sp.cfg.Hands.Left.Interface,
			ID:        fmt.Sprintf("0x%X", leftID),
			Data:      leftFrame,
		},
		{
			Interface: sp.cfg.Hands.Right.Interface,
			ID:        fmt.Sprintf("0x%X", rightID),
			Data:      rightFrame,
		},
	}, nil
}

// buildReleaseFrames 构建释放手指的CAN帧
func (sp *SequencePreprocessor) buildReleaseFrames() []ExecCANFrame {
	fingeringBuilder := NewFingeringBuilder()
	utils := NewUtils()

	var leftRelease, rightRelease []int
	if sp.instrument == "sn" {
		leftRelease = sp.cfg.SnLeftReleaseProfile
		rightRelease = sp.cfg.SnRightReleaseProfile
	} else {
		leftRelease = sp.cfg.SksLeftReleaseProfile
		rightRelease = sp.cfg.SksRightReleaseProfile
	}

	leftFrame := fingeringBuilder.BuildReleaseFrame(leftRelease)
	rightFrame := fingeringBuilder.BuildReleaseFrame(rightRelease)

	leftID := utils.ParseCanID(sp.cfg.Hands.Left.ID)
	rightID := utils.ParseCanID(sp.cfg.Hands.Right.ID)

	return []ExecCANFrame{
		{
			Interface: sp.cfg.Hands.Left.Interface,
			ID:        fmt.Sprintf("0x%X", leftID),
			Data:      leftFrame,
		},
		{
			Interface: sp.cfg.Hands.Right.Interface,
			ID:        fmt.Sprintf("0x%X", rightID),
			Data:      rightFrame,
		},
	}
}

// saveSequence 保存执行序列到文件
func (sp *SequencePreprocessor) saveSequence(sequence *ExecutionSequence, outputFile string) error {
	data, err := json.MarshalIndent(sequence, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %v", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
