# 逻辑接口映射优化

## 问题描述

### 原始设计的问题

**之前的 exec.json 文件：**
```json
{
  "frames": [
    {
      "i": "can3",        ← 硬编码的CAN接口
      "id": "0x28",
      "d": "..."
    },
    {
      "i": "can2",        ← 硬编码的CAN接口
      "id": "0x27",
      "d": "..."
    }
  ]
}
```

**问题：**
1. ❌ exec 文件与硬件配置强耦合
2. ❌ 更换硬件需要重新生成所有 exec 文件
3. ❌ 无法在不同设备间共享 exec 文件
4. ❌ 配置灵活性差

## 解决方案

### 新设计：逻辑标识 + 配置映射

**新的 exec.json 文件：**
```json
{
  "frames": [
    {
      "hand": "left",     ← 逻辑标识
      "id": "0x28",
      "d": "..."
    },
    {
      "hand": "right",    ← 逻辑标识
      "id": "0x27",
      "d": "..."
    }
  ]
}
```

**映射流程：**
```
exec.json                config.yaml              实际硬件
   ↓                        ↓                        ↓
hand: "left"    →    Hands.Left.Interface    →    can3
hand: "right"   →    Hands.Right.Interface   →    can2
```

**优势：**
1. ✅ exec 文件与硬件配置解耦
2. ✅ 更换硬件只需修改 config.yaml
3. ✅ 可以在不同设备间共享 exec 文件
4. ✅ 配置灵活性高

## 实现细节

### 1. 数据结构修改

**execution_types.go：**
```go
// ExecCANFrame 执行用CAN帧
type ExecCANFrame struct {
    Hand string `json:"hand"`  // 逻辑标识：left/right
    ID   string `json:"id"`    // 设备ID
    Data []byte `json:"d"`     // 数据
}
```

**对比：**
```go
// 之前
type ExecCANFrame struct {
    Interface string `json:"i"`  // 硬编码接口
    ID        string `json:"id"`
    Data      []byte `json:"d"`
}
```

### 2. 预处理器修改

**preprocessor.go：**
```go
// 生成时使用逻辑标识
return []ExecCANFrame{
    {
        Hand: "left",  // 逻辑标识
        ID:   fmt.Sprintf("0x%X", leftID),
        Data: leftFrame,
    },
    {
        Hand: "right", // 逻辑标识
        ID:   fmt.Sprintf("0x%X", rightID),
        Data: rightFrame,
    },
}
```

**对比：**
```go
// 之前：直接使用 config 中的接口
return []ExecCANFrame{
    {
        Interface: sp.cfg.Hands.Left.Interface,  // 硬编码
        ID:        fmt.Sprintf("0x%X", leftID),
        Data:      leftFrame,
    },
}
```

### 3. 执行引擎修改

**execution_engine.go：**
```go
// 执行时根据 config 映射
func (ee *ExecutionEngine) sendSingleFrame(frame ExecCANFrame) {
    // 映射逻辑标识到实际接口
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
    
    // 使用映射后的接口发送
    ee.utils.SendCanFrameAsync(ee.cfg, canInterface, id, frame.Data)
}
```

**对比：**
```go
// 之前：直接使用 exec 文件中的接口
func (ee *ExecutionEngine) sendSingleFrame(frame ExecCANFrame) {
    // 直接使用硬编码的接口
    ee.utils.SendCanFrameAsync(ee.cfg, frame.Interface, id, frame.Data)
}
```

## 使用场景

### 场景1：标准配置

**config.yaml：**
```yaml
hands:
  left:
    interface: can3
    id: "0x28"
  right:
    interface: can2
    id: "0x27"
```

**exec.json（通用）：**
```json
{
  "frames": [
    {"hand": "left", "id": "0x28", "d": "..."},
    {"hand": "right", "id": "0x27", "d": "..."}
  ]
}
```

**执行时映射：**
- `hand: "left"` → `can3`
- `hand: "right"` → `can2`

### 场景2：更换硬件（只需修改 config）

**新硬件的 config.yaml：**
```yaml
hands:
  left:
    interface: can0    # 改为 can0
    id: "0x28"
  right:
    interface: can1    # 改为 can1
    id: "0x27"
```

**exec.json（无需修改）：**
```json
{
  "frames": [
    {"hand": "left", "id": "0x28", "d": "..."},    ← 仍然是 "left"
    {"hand": "right", "id": "0x27", "d": "..."}    ← 仍然是 "right"
  ]
}
```

**执行时映射：**
- `hand: "left"` → `can0`  ← 自动映射到新接口
- `hand: "right"` → `can1` ← 自动映射到新接口

### 场景3：多设备共享 exec 文件

**设备A（树莓派1）：**
```yaml
# config.yaml
hands:
  left:
    interface: can3
```

**设备B（树莓派2）：**
```yaml
# config.yaml
hands:
  left:
    interface: can0    # 不同的接口
```

**两者都可以使用同一个 exec.json 文件！**

## 兼容性

### 向后兼容性

**旧格式（不兼容）：**
```json
{
  "i": "can3",
  "id": "0x28"
}
```

**新格式：**
```json
{
  "hand": "left",
  "id": "0x28"
}
```

**注意：** 
- ❌ 旧的 exec 文件**不能**用新程序播放（字段不匹配）
- ✅ 新的 exec 文件只能用新程序播放
- ⚠️ 需要重新生成所有 exec 文件

### 迁移步骤

1. **备份旧文件**
   ```bash
   mkdir -p backup/exec
   cp exec/*.exec.json backup/exec/
   ```

2. **更新程序**
   ```bash
   go build -o newsksgo
   ```

3. **重新生成所有 exec 文件**
   ```bash
   # 删除旧文件
   rm exec/*.exec.json
   
   # 批量重新生成
   ./batch_preprocess.sh sn 20
   ```

4. **验证新文件**
   ```bash
   # 查看新格式
   jq '.events[0].frames[0]' exec/test_sn_108_20.exec.json
   
   # 应该看到：
   # {
   #   "hand": "left",
   #   "id": "0x28",
   #   "d": "..."
   # }
   ```

5. **测试播放**
   ```bash
   ./newsksgo -json exec/test_sn_108_20.exec.json
   ```

## 优势总结

### 1. 灵活性

| 方面 | 旧设计 | 新设计 |
|------|--------|--------|
| **更换硬件** | 重新生成所有文件 | 只改 config.yaml |
| **多设备部署** | 每个设备不同文件 | 共享同一个文件 |
| **配置管理** | 分散在 exec 文件中 | 集中在 config.yaml |

### 2. 可维护性

**旧设计：**
```
硬件变化 → 修改代码 → 重新生成 → 重新部署
```

**新设计：**
```
硬件变化 → 修改 config.yaml → 重启程序
```

### 3. 可扩展性

未来可以轻松支持：
- 不同的手部配置（3只手？）
- 不同的CAN总线拓扑
- 动态设备映射
- 热插拔支持

### 4. 文件大小

**影响：** 几乎无变化
```
"i": "can3"   vs   "hand": "left"
```
都是短字符串，大小相当

## 测试验证

### 1. 单元测试（建议添加）

```go
func TestHandMapping(t *testing.T) {
    cfg := Config{
        Hands: struct {
            Left  HandConfig
            Right HandConfig
        }{
            Left:  HandConfig{Interface: "can3"},
            Right: HandConfig{Interface: "can2"},
        },
    }
    
    engine := &ExecutionEngine{cfg: cfg}
    
    // 测试左手映射
    frame := ExecCANFrame{Hand: "left"}
    interface := engine.getInterface(frame)
    assert.Equal(t, "can3", interface)
    
    // 测试右手映射
    frame = ExecCANFrame{Hand: "right"}
    interface = engine.getInterface(frame)
    assert.Equal(t, "can2", interface)
}
```

### 2. 集成测试

```bash
# 1. 生成测试文件
./newsksgo -preprocess -in trsmusic/test.json -instrument sn -bpm 108 -tongue 20

# 2. 检查格式
jq '.events[0].frames[0]' exec/test_sn_108_20.exec.json
# 应该看到 "hand": "left"

# 3. 修改配置测试映射
# 修改 config.yaml 中的接口配置
# 观察播放是否使用新接口

# 4. 验证播放
./newsksgo -json exec/test_sn_108_20.exec.json
```

## 最佳实践

### 1. 配置管理

```yaml
# config.yaml
# 生产环境
hands:
  left:
    interface: can3
    id: "0x28"
  right:
    interface: can2
    id: "0x27"

# 开发环境
# hands:
#   left:
#     interface: can0
#     id: "0x28"
#   right:
#     interface: can1
#     id: "0x27"
```

### 2. exec 文件命名

文件名不需要包含接口信息：
```
✅ 青花瓷-葫芦丝-4min-108_sn_108_20.exec.json
❌ 青花瓷-葫芦丝-4min-108_sn_108_20_can3_can2.exec.json
```

### 3. 部署流程

```bash
# 1. 生成 exec 文件（一次）
./batch_preprocess.sh sn 20

# 2. 复制到所有设备（exec 文件通用）
scp exec/*.exec.json pi@device1:/home/pi/sksgo/exec/
scp exec/*.exec.json pi@device2:/home/pi/sksgo/exec/

# 3. 每个设备只需配置自己的 config.yaml
ssh pi@device1 "vi /home/pi/sksgo/config.yaml"
ssh pi@device2 "vi /home/pi/sksgo/config.yaml"
```

## 故障排查

### 问题1：播放无声音

**可能原因：** config.yaml 中的接口配置错误

**检查：**
```bash
# 查看 config
cat config.yaml | grep -A 3 "hands:"

# 查看 exec 文件
jq '.events[0].frames[0]' exec/test_sn_108_20.exec.json

# 确认映射
# hand: "left" → config.hands.left.interface
```

### 问题2：找不到手部标识

**错误信息：** `⚠️  警告: 未知的手部标识: xxx`

**原因：** exec 文件格式错误或损坏

**解决：**
```bash
# 重新生成 exec 文件
./newsksgo -preprocess -in trsmusic/xxx.json -instrument sn -bpm 108 -tongue 20
```

### 问题3：旧文件无法播放

**错误：** 无反应或字段解析错误

**原因：** 使用了旧格式的 exec 文件

**解决：**
```bash
# 删除旧文件
rm exec/*.exec.json

# 重新生成所有文件
./batch_preprocess.sh sn 20
```

## 总结

### 关键改进

1. ✅ **解耦设计** - exec 文件与硬件配置分离
2. ✅ **配置灵活** - 通过 config.yaml 动态映射
3. ✅ **易于部署** - 多设备共享 exec 文件
4. ✅ **易于维护** - 集中管理硬件配置

### 需要注意

1. ⚠️ **不兼容旧格式** - 需要重新生成所有 exec 文件
2. ⚠️ **配置正确性** - 确保 config.yaml 中的接口配置正确
3. ⚠️ **测试验证** - 更新后需要充分测试

### 下一步

1. 重新编译程序
2. 重新生成所有 exec 文件
3. 测试不同配置的兼容性
4. 更新部署文档

