# 吐音效果优化说明

## 问题描述

**原始问题：**
1. 相同音符连接时吐音效果不自然
2. 气泵控制过于频繁（每个音符都发送"on"命令）
3. 不同音符切换时不应该控制气泵

## 优化方案

### 核心原则

```
气泵控制原则：
1. ✅ 第一个音符：开启气泵
2. ✅ 不同音符切换：不控制气泵（保持开启）
3. ✅ 相同音符连接：关闭 → 开启（吐音效果）
4. ✅ 休止符：关闭气泵
5. ✅ 演奏结束：关闭气泵
```

### 优化前后对比

#### 优化前（❌ 问题）

```
音符序列: Do - Do - Re - Mi - Mi - Mi - NO - Fa

生成的事件:
Do  → 指法 + SerialCmd:"on"   ← 开启
Do  → 无指法 + SerialCmd:"on"  ← 重复开启（不必要）
Re  → 指法 + SerialCmd:"on"   ← 重复开启（不必要）
Mi  → 指法 + SerialCmd:"on"   ← 重复开启（不必要）
Mi  → 无指法 + SerialCmd:"on"  ← 重复开启（不必要）
Mi  → 无指法 + SerialCmd:"on"  ← 重复开启（不必要）
NO  → 释放 + SerialCmd:"off"  ← 关闭
Fa  → 指法 + SerialCmd:"on"   ← 开启

问题：每个音符都发送"on"，气泵可能频繁响应
```

#### 优化后（✅ 正确）

```
音符序列: Do - Do - Re - Mi - Mi - Mi - NO - Fa

生成的事件:
Do  → 指法 + SerialCmd:"on"   ← 第一个音符，开启
关  → 无帧 + SerialCmd:"off"  ← 吐音间隙
Do  → 无帧 + SerialCmd:"on"   ← 吐音续接
Re  → 指法 + SerialCmd:""    ← 不同音符，不控制气泵（保持开启）
Mi  → 指法 + SerialCmd:""    ← 不同音符，不控制气泵（保持开启）
关  → 无帧 + SerialCmd:"off"  ← 吐音间隙
Mi  → 无帧 + SerialCmd:"on"   ← 吐音续接
关  → 无帧 + SerialCmd:"off"  ← 吐音间隙
Mi  → 无帧 + SerialCmd:"on"   ← 吐音续接
NO  → 释放 + SerialCmd:"off"  ← 休止符，关闭
Fa  → 指法 + SerialCmd:"on"   ← 空拍后第一个音符，开启

优点：
1. 不同音符切换不控制气泵（自然）
2. 相同音符有明确的关-开序列（吐音清晰）
3. 减少不必要的串口通信
```

## 代码修改详情

### 1. 添加 isFirstNote 标记

```go
// preprocessor.go:117
isFirstNote := true  // 标记是否为第一个音符（需要开启气泵）

// 空拍后重置
if event.Note == "NO" {
    // ...
    isFirstNote = true  // 空拍后下一个音符需要开启气泵
}

// 音符播放后设置为false
if !prevIsSame {
    // ...
    isFirstNote = false  // 已开启气泵
}
```

### 2. 修改 generateNormalEvent 函数

```go
// 只有第一个音符或空拍后需要开启气泵
serialCmd := ""
if isFirstNote {
    serialCmd = "on"
}

return ExecutionEvent{
    // ...
    SerialCmd: serialCmd,  // 只有第一个音符才发送"on"，其他时候为空
}
```

### 3. 修改 generateTonguingStart 函数

```go
// 只有第一个音符或空拍后需要开启气泵，其他时候不控制气泵
serialCmd := ""
if isFirstNote {
    serialCmd = "on"
}

events = append(events, ExecutionEvent{
    // ...
    SerialCmd: serialCmd,  // 可能为空
})
```

### 4. 执行引擎检查

```go
// execution_engine.go:177
// 已有检查，无需修改
if event.SerialCmd != "" {
    go ee.sendSerialCmd(event.SerialCmd)
}
if len(event.Frames) > 0 {
    for _, frame := range event.Frames {
        go ee.sendSingleFrame(frame)
    }
}
```

## 优化效果分析

### 1. 串口通信减少

**示例曲目：100个音符**

| 场景 | 优化前 | 优化后 | 减少 |
|------|--------|--------|------|
| 全部不同音符 | 100次 on | 1次 on + 1次 off | ~98% |
| 50%相同音符连接 | 100次 on | 26次 (on+off交替) | ~74% |
| 连续相同音符×10 | 10次 on | 1次 on + 18次 (on+off) | ~0% |

**结论：**
- 不同音符为主的曲目：减少90%+的串口通信
- 相同音符较多的曲目：通信量相当，但逻辑更清晰

### 2. 吐音效果改善

**吐音效果流程：**

```
优化前（可能不清晰）：
音符A(on) → 延迟 → 音符A(on) → ...
           ↑ 可能没有明确的关闭

优化后（清晰明确）：
音符A(on) → 延迟(off) → 音符A(on) → ...
           ↑ 明确的关-开序列
```

### 3. 演奏状态管理

```
气泵状态追踪更清晰：
- 第一个音符：off → on
- 不同音符切换：保持 on
- 相同音符连接：on → off → on (吐音)
- 休止符：on → off
- 演奏结束：on → off
```

## 进一步优化建议

### 1. 添加气泵状态跟踪（可选）

```go
// 在执行引擎中追踪气泵状态，避免重复命令
type ExecutionEngine struct {
    // ...
    pumpState bool  // 当前气泵状态
}

func (ee *ExecutionEngine) sendSerialCmd(cmd string) {
    switch cmd {
    case "on":
        if !ee.pumpState {  // 只在关闭时才开启
            GlobalPumpOn()
            ee.pumpState = true
        }
    case "off":
        if ee.pumpState {  // 只在开启时才关闭
            GlobalPumpOff()
            ee.pumpState = false
        }
    }
}
```

### 2. 吐音延迟自适应（可选）

根据音符时长动态调整吐音延迟：

```go
// 短音符使用较短的吐音延迟
tonguingDelay := sp.tonguingDelay
if durationMS < 200 {  // 小于200ms的音符
    tonguingDelay = int(float64(tonguingDelay) * 0.7)  // 减少30%
}
```

### 3. 监控和日志（可选）

添加详细的执行日志：

```go
// 记录气泵控制次数
type ExecutionStats struct {
    PumpOnCount  int
    PumpOffCount int
    FrameCount   int
}

// 在执行结束时输出
fmt.Printf("📊 执行统计:\n")
fmt.Printf("   气泵开启: %d次\n", stats.PumpOnCount)
fmt.Printf("   气泵关闭: %d次\n", stats.PumpOffCount)
fmt.Printf("   CAN帧数: %d\n", stats.FrameCount)
```

### 4. 预切换优化（已实现）

空拍时预切换下一个音符的指法：

```go
// preprocessor.go:320 已实现
// 在空拍结束前20%时预切换指法
events = append(events, ExecutionEvent{
    TimestampMS: timestampMS + durationMS*0.8,
    DurationMS:  durationMS * 0.2,
    Note:        fmt.Sprintf("PRE_%s", allEvents[nextIndex].Note),
    Frames:      nextFingeringFrames,
    SerialCmd:   "",  // 不控制气泵
})
```

## 测试建议

### 1. 单元测试

测试不同音符序列生成的事件：

```go
// 测试用例
testCases := []struct{
    notes    []string
    expected []string  // 期望的 SerialCmd 序列
}{
    {
        notes:    []string{"Do", "Re", "Mi"},
        expected: []string{"on", "", ""},  // 只有第一个"on"
    },
    {
        notes:    []string{"Do", "Do", "Do"},
        expected: []string{"on", "off", "on", "off", "on"},  // 吐音序列
    },
    {
        notes:    []string{"Do", "NO", "Mi"},
        expected: []string{"on", "off", "on"},  // 空拍后重新开启
    },
}
```

### 2. 实际演奏测试

```bash
# 生成优化后的exec文件
./newsksgo -preprocess -in trsmusic/test.json -instrument sn -bpm 108 -tongue 20

# 播放测试
./newsksgo -json exec/test_sn_108_20.exec.json

# 观察：
# 1. 相同音符连接是否有明显的吐音效果
# 2. 不同音符切换是否自然流畅
# 3. 气泵响应是否平稳
```

### 3. 性能测试

```bash
# 测试长曲目的执行效率
./newsksgo -json exec/long_song_sn_120_20.exec.json

# 监控：
# 1. CPU使用率
# 2. 串口通信频率
# 3. 时序精度
```

## 总结

### 关键改进

1. ✅ **减少不必要的串口通信** - 不同音符切换时不控制气泵
2. ✅ **吐音效果更清晰** - 明确的关-开序列
3. ✅ **逻辑更清晰** - 气泵状态管理更明确
4. ✅ **代码可维护性提升** - 意图更清晰

### 影响范围

- ✅ 预处理器 (`preprocessor.go`)
- ✅ 执行引擎 (`execution_engine.go`) - 无需修改，已有检查
- ✅ 生成的 exec 文件格式 - 兼容，只是 SerialCmd 可能为空

### 兼容性

- ✅ 旧的 exec 文件仍可正常播放
- ✅ 新的 exec 文件包含更优化的控制逻辑
- ✅ 建议重新预处理所有音乐文件以获得最佳效果

### 下一步

1. 编译新版本：`go build -o newsksgo`
2. 重新预处理音乐文件：`./batch_preprocess.sh sn 20`
3. 测试播放效果
4. 根据实际效果微调吐音延迟参数


