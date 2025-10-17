# 吐音逻辑修正说明

## 问题描述

之前的实现存在两个关键问题：

1. **没有实际的吐音间隙 (gap)**：只关闭了气泵，但没有 `time.Sleep(tongue_ms)` 的等待时间，导致没有"断气"效果
2. **下一个音符未扣除份额 (gR)**：只在当前音符扣除了 gL，导致两个相同音符的总时长减少了 gL，使BPM被"加快"

## 正确逻辑

### 相同音符之间的吐音处理

对于两个相同的音符 A 和 B：

```
原始时长：
- A 音符：base_A = (duration_A / BPM) * 60 * 1000 ms
- B 音符：base_B = (duration_B / BPM) * 60 * 1000 ms
- 总时长：base_A + base_B

实际演奏时长：
- gL = tongue_ms × (duration_A / (duration_A + duration_B))  // A承担的补偿
- gR = tongue_ms × (duration_B / (duration_A + duration_B))  // B承担的补偿

时间轴：
1. A 播放 (base_A - gL) ms，气泵开启
2. 关闭气泵
3. 等待 tongue_ms ms（实际吐音间隙）
4. B 播放 (base_B - gR) ms，气泵开启

总时长 = (base_A - gL) + tongue_ms + (base_B - gR)
       = base_A + base_B + tongue_ms - (gL + gR)
       = base_A + base_B + tongue_ms - tongue_ms
       = base_A + base_B  ✓ 保持原始总时长
```

### 不同音符之间

```
正常播放，无补偿：
1. A 播放 base_A ms，气泵开启
2. B 播放 base_B ms，气泵保持开启（指法切换，但不断气）
```

## 代码实现

### 关键变量

```go
skipNextCompensation := false  // 标记下一个音符是否需要跳过时间补偿
nextCompensation := 0.0        // 下一个音符需要扣除的时间（毫秒）
```

### 当前音符与下一个音符相同

```go
// 计算时间补偿
gL := float64(pe.tonguingDelay) * (currentDuration / totalDuration)
gR := float64(pe.tonguingDelay) * (nextDuration / totalDuration)

// 当前音符播放 (base - gL)
playDuration = baseDuration - time.Duration(gL)*time.Millisecond
utils.ControlAirPumpWithLock(pe.cfg, true)
time.Sleep(playDuration)

// 关闭气泵
utils.ControlAirPumpWithLock(pe.cfg, false)

// *** 关键：插入实际的吐音间隙 ***
time.Sleep(time.Duration(pe.tonguingDelay) * time.Millisecond)

// 标记下一个音符需要扣除 gR
skipNextCompensation = true
nextCompensation = gR
```

### 下一个音符开始时

```go
// 如果这个音符需要扣除上一次计算的补偿时间
if skipNextCompensation && nextCompensation > 0 {
    playDuration = baseDuration - time.Duration(nextCompensation)*time.Millisecond
    skipNextCompensation = false
    nextCompensation = 0.0
}
```

## 示例

假设：
- BPM = 120
- tongue_ms = 30ms
- 两个相同音符 A4，每个 1 拍

计算：
```
base_A = base_B = (1.0 / 120) × 60 × 1000 = 500ms
gL = gR = 30 × (1.0 / 2.0) = 15ms

实际演奏：
1. A4 播放：500 - 15 = 485ms（气泵开）
2. 关闭气泵
3. 等待：30ms（吐音间隙）
4. A4 播放：500 - 15 = 485ms（气泵开）

总时长 = 485 + 30 + 485 = 1000ms = 1秒 ✓
正好对应 120 BPM 下的 2 拍
```

## 时间轴可视化

```
当前音符 A        吐音间隙        下一个音符 B
|--base_A-gL--|  |--tongue--|  |--base_B-gR--|
|   485ms     |  |  30ms    |  |   485ms     |
[气泵开]        [气泵关]        [气泵开]
```

## 特殊情况处理

### 休止符（NO）

```go
if event.Note == "NO" {
    // 关闭气泵 + 松开手指
    // 重置补偿标记
    skipNextCompensation = false
    nextCompensation = 0.0
    continue
}
```

### 短音符

如果 `base - g < 0`，设置为 0：

```go
if playDuration < 0 {
    playDuration = 0
}
```

## 优点

1. ✅ 保持精确的 BPM，总时长不变
2. ✅ 真实的吐音效果（有实际的断气时间）
3. ✅ 按比例公平分配吐音延迟
4. ✅ 支持前端动态调整吐音延迟时间

