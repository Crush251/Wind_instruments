# 实用重构方案（简化版）

## 🎯 设计原则

**小项目原则：**
- ✅ **适度模块化**：不要过度拆分
- ✅ **文件合并**：相关功能放在同一文件
- ✅ **扁平结构**：减少目录层级
- ✅ **实用优先**：保持简单可维护

---

## 📁 推荐的目录结构（简化版）

```
sksgo/
├── cmd/
│   └── sksgo/
│       └── main.go              # 入口：参数解析、启动
│
├── internal/                    # 内部包（只有一层，不细分）
│   ├── models.go               # 所有类型定义（合并types.go + execution_types.go + constants.go）
│   ├── config.go               # 配置加载（从file_reader.go提取）
│   ├── hardware.go             # 硬件控制（合并：气泵 + CAN + 指法构建 + 预备手势）
│   ├── music.go                # 音乐处理（合并：文件读取 + 扫描 + 解析）
│   ├── processor.go            # 预处理器（preprocessor.go重命名）
│   ├── executor.go             # 执行引擎（execution_engine.go重命名）
│   └── server.go               # Web服务（web_server.go重命名）
│
├── web/                        # Web前端（不变）
│   ├── static/
│   └── templates/
│
├── config/                     # 配置文件（不变）
├── trsmusic/                   # 音乐文件（不变）
├── exec/                       # 执行序列（不变）
│
├── go.mod
├── go.sum
└── README.md
```

### 文件数量对比

| 方案 | 文件数量 | 目录层级 | 复杂度 |
|------|---------|---------|--------|
| **当前** | 13个Go文件 | 1层 | 混乱 |
| **过度细化** | 30+个文件 | 3-4层 | 过高 |
| **简化版** | 9个Go文件 | 2层 | ✅ 适中 |

---

## 📝 文件职责说明

### 1. `cmd/sksgo/main.go` - 程序入口
```go
// 职责：
// - 解析命令行参数
// - 初始化全局资源（气泵控制器）
// - 路由到不同模式（预处理/执行/Web服务）
```

**迁移：** `main.go` → `cmd/sksgo/main.go`

---

### 2. `internal/models.go` - 类型定义（合并3个文件）
```go
// 合并内容：
// - types.go（Config, TimelineFile, FingeringEntry等）
// - execution_types.go（ExecutionSequence, ExecutionEvent等）
// - constants.go（常量定义）

// 职责：
// - 所有数据结构的定义
// - 全局常量
// - 全局变量声明（playbackController等）
```

**迁移：**
- `types.go` + `execution_types.go` + `constants.go` → `internal/models.go`

---

### 3. `internal/config.go` - 配置管理
```go
// 职责：
// - 加载YAML配置文件
// - 配置验证和默认值
// - 配置相关的工具函数

// 从 file_reader.go 提取配置相关代码
```

**迁移：** `file_reader.go` 中的 `LoadConfig()` → `internal/config.go`

---

### 4. `internal/hardware.go` - 硬件控制（合并4个文件）
```go
// 合并内容：
// - utils.go 中的气泵控制（GlobalPumpOn/Off等）
// - utils.go 中的CAN控制（SendCanFrame等）
// - fingering_builder.go（指法构建）
// - ready_gesture.go（预备手势）

// 职责：
// - 气泵控制器（初始化、命令发送）
// - CAN总线通信（HTTP客户端）
// - 指法构建（生成CAN帧）
// - 预备手势执行

// 文件结构：
// - PumpController 结构体和相关函数
// - CANClient 结构体和相关函数
// - FingeringBuilder 结构体和相关函数
// - ReadyGestureController 结构体和相关函数
```

**迁移：**
- `utils.go` 中的硬件相关代码 → `internal/hardware.go`
- `fingering_builder.go` → `internal/hardware.go`
- `ready_gesture.go` → `internal/hardware.go`

---

### 5. `internal/music.go` - 音乐处理（合并3个文件）
```go
// 合并内容：
// - file_reader.go（除了配置加载）
// - music_scanner.go（文件扫描）
// - preprocessor.go 中的 parseTimeline

// 职责：
// - 读取音乐文件（JSON）
// - 读取指法映射文件（YAML）
// - 扫描音乐文件列表
// - 解析时间轴数据

// 文件结构：
// - FileReader 结构体
// - MusicFileScanner 结构体
// - 文件读取相关函数
```

**迁移：**
- `file_reader.go`（除配置外） → `internal/music.go`
- `music_scanner.go` → `internal/music.go`

---

### 6. `internal/processor.go` - 预处理器
```go
// 职责：
// - 解析音乐时间轴
// - 生成执行序列（exec.json）
// - 处理吐音逻辑
// - 计算时间补偿

// 文件结构：
// - SequencePreprocessor 结构体
// - 预处理相关函数
```

**迁移：** `preprocessor.go` → `internal/processor.go`

---

### 7. `internal/executor.go` - 执行引擎
```go
// 职责：
// - 加载执行序列文件
// - 执行播放逻辑
// - 时间控制
// - 进度更新

// 文件结构：
// - ExecutionEngine 结构体
// - 播放相关函数
```

**迁移：** `execution_engine.go` → `internal/executor.go`

---

### 8. `internal/server.go` - Web服务
```go
// 职责：
// - HTTP服务器
// - API路由和处理器
// - 静态文件服务
// - 模板渲染

// 文件结构：
// - WebServer 结构体
// - API处理函数
// - 路由设置
```

**迁移：** `web_server.go` → `internal/server.go`

---

### 9. 删除/合并的文件

**删除：**
- `cli_executor.go` → 合并到 `cmd/sksgo/main.go`（逻辑简单）

**合并：**
- `utils.go` → 拆分为 `internal/hardware.go` 和 `internal/config.go`

---

## 🔄 迁移步骤（简化版）

### 阶段1：创建基础结构（5分钟）

```bash
# 1. 创建目录
mkdir -p cmd/sksgo
mkdir -p internal

# 2. 移动main.go
mv main.go cmd/sksgo/main.go

# 3. 创建占位文件
touch internal/models.go
touch internal/config.go
touch internal/hardware.go
touch internal/music.go
touch internal/processor.go
touch internal/executor.go
touch internal/server.go
```

### 阶段2：合并类型定义（10分钟）

```bash
# 合并3个文件到1个
# types.go + execution_types.go + constants.go → internal/models.go

# 更新import：所有文件中的 import 改为
import "sksgo/internal"
```

### 阶段3：迁移文件（逐步进行）

```bash
# 1. 迁移配置
# file_reader.go 的 LoadConfig() → internal/config.go

# 2. 迁移硬件控制
# utils.go + fingering_builder.go + ready_gesture.go → internal/hardware.go

# 3. 迁移音乐处理
# file_reader.go（剩余） + music_scanner.go → internal/music.go

# 4. 重命名文件
mv preprocessor.go internal/processor.go
mv execution_engine.go internal/executor.go
mv web_server.go internal/server.go
```

### 阶段4：清理

```bash
# 删除旧文件
rm types.go execution_types.go constants.go
rm utils.go file_reader.go music_scanner.go
rm fingering_builder.go ready_gesture.go
rm cli_executor.go
```

---

## 📊 最终结构对比

### 当前结构（13个文件）
```
├── main.go
├── types.go
├── execution_types.go
├── constants.go
├── utils.go
├── file_reader.go
├── music_scanner.go
├── fingering_builder.go
├── ready_gesture.go
├── preprocessor.go
├── execution_engine.go
├── web_server.go
└── cli_executor.go
```

### 简化后结构（9个文件）
```
├── cmd/sksgo/
│   └── main.go
└── internal/
    ├── models.go          # 合并了3个文件
    ├── config.go          # 从file_reader.go提取
    ├── hardware.go        # 合并了4个文件
    ├── music.go           # 合并了2个文件
    ├── processor.go
    ├── executor.go
    └── server.go
```

**文件减少：** 13个 → 9个（减少30%）

---

## ✅ 优势

### 1. 保持简单
- ✅ 只有2层目录（cmd/ 和 internal/）
- ✅ 文件数量适中（9个Go文件）
- ✅ 每个文件职责清晰

### 2. 适度模块化
- ✅ 类型定义集中（models.go）
- ✅ 硬件控制集中（hardware.go）
- ✅ 音乐处理集中（music.go）

### 3. 易于维护
- ✅ 相关功能在一起
- ✅ 减少文件跳转
- ✅ 降低认知负担

### 4. 符合Go习惯
- ✅ 遵循 `cmd/` 和 `internal/` 约定
- ✅ 保持包名简洁
- ✅ 便于后续扩展

---

## ⚠️ 注意事项

### 1. 文件大小控制
- 如果某个文件超过500行，可以拆分
- 例如：`hardware.go` 太大时，可以拆分为 `hardware_pump.go` 和 `hardware_can.go`

### 2. 全局变量处理
- `globalPumpController` → 放在 `internal/hardware.go`
- `playbackController` → 放在 `internal/models.go`
- `globalHTTPClient` → 放在 `internal/hardware.go`

### 3. 导入路径
```go
// 所有文件统一使用
import "sksgo/internal"
```

### 4. 测试文件
```
internal/
├── models.go
├── models_test.go      # 对应测试
├── hardware.go
├── hardware_test.go    # 对应测试
└── ...
```

---

## 🚀 快速开始

### 最小改动版本（推荐先做这个）

如果不想大改，可以先做最小改动：

```bash
# 1. 只创建cmd目录
mkdir -p cmd/sksgo
mv main.go cmd/sksgo/main.go

# 2. 合并类型定义文件
cat types.go execution_types.go constants.go > internal/models.go

# 3. 更新go.mod
# 在go.mod中添加：
# module sksgo

# 4. 更新import
# 所有文件添加：import "sksgo/internal"
```

这样可以先解决最核心的问题（类型定义分散），其他慢慢迁移。

---

## 📈 渐进式迁移策略

### 第1周：基础结构
- ✅ 创建 `cmd/sksgo/`
- ✅ 合并类型定义到 `internal/models.go`
- ✅ 验证编译通过

### 第2周：硬件控制
- ✅ 迁移硬件相关代码到 `internal/hardware.go`
- ✅ 删除 `utils.go`, `fingering_builder.go`, `ready_gesture.go`

### 第3周：其他模块
- ✅ 迁移音乐处理到 `internal/music.go`
- ✅ 迁移Web服务到 `internal/server.go`
- ✅ 完成清理

---

## 🎯 总结

**这个方案的优势：**
- ✅ **文件数量适中**：9个文件，易于管理
- ✅ **结构清晰**：2层目录，职责明确
- ✅ **迁移简单**：主要是合并和重命名
- ✅ **保持实用**：不过度工程化

**适合场景：**
- ✅ 小型项目（< 20个文件）
- ✅ 团队规模小（1-3人）
- ✅ 需要适度重构但不想过度设计

**不适合场景：**
- ❌ 大型项目（> 50个文件）
- ❌ 需要严格的模块隔离
- ❌ 需要支持多版本API

---

这个方案更实用，你觉得如何？需要我帮你开始迁移吗？


