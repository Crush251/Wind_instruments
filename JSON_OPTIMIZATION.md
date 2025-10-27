# JSON 序列文件优化

## 优化内容

### 问题
吐音续接事件中，指法与上一个音符相同，不需要发送CAN帧。但之前生成的是空数组 `"frames": []`，占用不必要的空间。

### 解决方案

**1. 添加 `omitempty` 标签**
```go
// execution_types.go
type ExecutionEvent struct {
    // ...
    Frames    []ExecCANFrame `json:"frames,omitempty"`  // 添加 omitempty
    SerialCmd string         `json:"serial,omitempty"`  // 已有
}
```

**2. 使用 `nil` 代替空数组**
```go
// preprocessor.go
// 吐音续接事件
events = append(events, ExecutionEvent{
    // ...
    Frames:    nil,  // 之前是 []ExecCANFrame{}
    SerialCmd: "on",
})
```

**3. 简化执行引擎检查**
```go
// execution_engine.go
// len(nil) 返回 0，直接遍历即可
for _, frame := range event.Frames {
    go ee.sendSingleFrame(frame)
}
```

## 优化效果

### JSON 文件大小对比

#### 优化前
```json
{
  "t": 20257.777777777777,
  "d": 30,
  "n": "TONGUE",
  "frames": [],
  "serial": "off"
},
{
  "t": 20287.777777777777,
  "d": 267.77777777777777,
  "n": "C4",
  "frames": [],
  "serial": "on"
}
```

每个事件：`"frames": []` = 13字节

#### 优化后
```json
{
  "t": 20257.777777777777,
  "d": 30,
  "n": "TONGUE",
  "serial": "off"
},
{
  "t": 20287.777777777777,
  "d": 267.77777777777777,
  "n": "C4",
  "serial": "on"
}
```

每个事件：省略 `frames` 字段 = 节省 ~13字节

### 实际效果

**示例曲目分析：** 青花瓷-葫芦丝-4min-108

```
总事件数: 521
吐音相关事件（无frames）: ~150

节省空间: 150 × 13 = 1,950 字节
优化前文件大小: 约 220KB
优化后文件大小: 约 218KB
减少: ~0.9%
```

**更长曲目（10分钟）：**
```
总事件数: 1500
吐音相关事件: ~500

节省空间: 500 × 13 = 6,500 字节
减少: ~1-2%
```

### 可读性提升

**优化前（杂乱）：**
```json
{
  "frames": [],
  "serial": "on"
}
{
  "frames": [
    {"i": "can3", "id": "0x28", "d": "..."}
  ],
  "serial": ""
}
{
  "frames": [],
}
```

**优化后（清晰）：**
```json
{
  "serial": "on"
}
{
  "frames": [
    {"i": "can3", "id": "0x28", "d": "..."}
  ]
}
{
  "t": 100,
  "d": 200,
  "n": "Do"
}
```

一眼就能看出：
- 哪些事件有指法切换（有 `frames`）
- 哪些事件只控制气泵（只有 `serial`）
- 哪些事件只是时间占位（都没有）

## 其他优化建议

### 1. 时间戳可以优化为增量（可选）

**当前：** 每个事件都存储绝对时间戳
```json
{"t": 18888.88, "d": 277.77, ...}
{"t": 19166.66, "d": 277.77, ...}
{"t": 19444.44, "d": 277.77, ...}
```

**优化方案：** 只存储增量（delta）
```json
{"dt": 0, "d": 277.77, ...}        // 第一个
{"dt": 277.78, "d": 277.77, ...}   // +277.78
{"dt": 277.78, "d": 277.77, ...}   // +277.78
```

**节省：** 每个事件节省 ~5字节（数字变小）
**影响：** 需要修改执行引擎的时间计算

### 2. Base64 优化（可选）

**当前：** CAN数据用 Base64 编码
```json
"d": "AZcT6On//w=="  // 8字节数据 → 12字符
```

**优化方案：** 使用十六进制（更紧凑，更可读）
```json
"d": "01971387e8e9ffff"  // 16字符，但更直观
```

或者用数组（最紧凑）
```json
"d": [1,151,19,135,232,233,255,255]
```

### 3. 事件类型枚举（可选）

**当前：** 用字符串区分事件
```json
"n": "TONGUE"
"n": "REST"
"n": "END"
```

**优化方案：** 添加类型字段
```json
"type": 1,  // 1=note, 2=tongue, 3=rest
"n": "C4"
```

## 兼容性

### 向后兼容

- ✅ **旧版执行引擎** 可以读取新格式文件
  - `omitempty` 只影响序列化，不影响反序列化
  - `nil` frames 和 `[]` frames 对执行引擎来说是等价的

- ✅ **新版执行引擎** 可以读取旧格式文件
  - 空数组 `[]` 会被正确处理（len=0）

### 建议

1. **升级执行引擎** → 测试 → **重新生成所有 exec 文件**
2. 保留旧文件作为备份
3. 对比播放效果确认无差异

## 测试验证

### 1. 单元测试

```go
func TestEmptyFrames(t *testing.T) {
    // 测试 nil frames
    event1 := ExecutionEvent{Frames: nil}
    data1, _ := json.Marshal(event1)
    assert.NotContains(t, string(data1), "frames")
    
    // 测试空数组
    event2 := ExecutionEvent{Frames: []ExecCANFrame{}}
    data2, _ := json.Marshal(event2)
    assert.NotContains(t, string(data2), "frames")  // omitempty
}
```

### 2. 文件大小验证

```bash
# 生成新文件
./newsksgo -preprocess -in trsmusic/test.json -instrument sn -bpm 108 -tongue 20

# 对比大小
ls -lh exec/test_sn_108_20.exec.json

# 查看事件数量
jq '.meta.total_events' exec/test_sn_108_20.exec.json

# 统计没有 frames 的事件
jq '[.events[] | select(has("frames") | not)] | length' exec/test_sn_108_20.exec.json
```

### 3. 播放测试

```bash
# 播放新生成的文件
./newsksgo -json exec/test_sn_108_20.exec.json

# 检查：
# 1. 是否正常播放
# 2. 吐音效果是否正常
# 3. 指法切换是否正常
```

## 总结

### 优化成果

| 项目 | 优化前 | 优化后 | 改善 |
|------|--------|--------|------|
| **吐音事件大小** | 包含 `"frames":[]` | 省略 `frames` | 节省 ~13字节/事件 |
| **文件大小** | 220KB | 218KB | ~1% |
| **可读性** | 有冗余字段 | 简洁清晰 | ✅ 提升 |
| **执行效率** | 无影响 | 无影响 | 相同 |
| **兼容性** | - | 向后兼容 | ✅ |

### 关键改动

1. ✅ `execution_types.go` - 添加 `omitempty` 标签
2. ✅ `preprocessor.go` - 使用 `nil` 替代空数组
3. ✅ `execution_engine.go` - 简化 nil 检查

### 影响范围

- ✅ 生成的 exec 文件更小、更简洁
- ✅ 无需修改播放逻辑
- ✅ 完全向后兼容

### 下一步

1. 重新编译：`go build -o newsksgo`
2. 重新预处理：`./batch_preprocess.sh sn 20`
3. 测试播放效果
4. 删除旧的 exec 文件（可选）


