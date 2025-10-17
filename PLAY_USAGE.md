# 演奏控制脚本使用说明

## 快速开始

### 1. 启动Web服务
```bash
cd /home/linkerhand/sks/sksgo
go run main.go
# 或使用编译后的可执行文件
./sksgo
```

服务将在 `http://localhost:8088` 上运行。

### 2. 使用脚本控制演奏

#### 开始演奏
```bash
./play.sh play <音乐文件> [乐器类型] [BPM] [吐音延迟]
```

**参数说明：**
- `<音乐文件>`：必需，音乐文件名（如 `test.json`）
- `[乐器类型]`：可选，`sks`(萨克斯) 或 `sn`(唢呐)，默认 `sks`
- `[BPM]`：可选，节拍速度，`0` 表示使用文件默认值，默认 `0`
- `[吐音延迟]`：可选，吐音延迟时间（毫秒），默认 `30`

#### 停止演奏
```bash
./play.sh stop
```

## 使用示例

### 示例1：使用默认设置演奏萨克斯
```bash
./play.sh play test.json
```
- 乐器：萨克斯（默认）
- BPM：使用文件中的BPM（默认）
- 吐音延迟：30ms（默认）

### 示例2：指定BPM演奏萨克斯
```bash
./play.sh play test.json sks 120 30
```
- 乐器：萨克斯
- BPM：120
- 吐音延迟：30ms

### 示例3：演奏唢呐
```bash
./play.sh play molihua.json sn 100 30
```
- 乐器：唢呐
- BPM：100
- 吐音延迟：30ms

### 示例4：停止当前演奏
```bash
./play.sh stop
```
不管当前在演奏什么，都会停止演奏并关闭气泵。

## 实际场景

### 场景1：测试演奏
```bash
# 开始演奏
./play.sh play test.json

# 等待10秒
sleep 10

# 停止演奏
./play.sh stop
```

### 场景2：连续演奏多首曲目
```bash
#!/bin/bash
# 演奏萨克斯曲目
./play.sh play test.json sks 120 30
sleep 30  # 播放30秒

./play.sh stop
sleep 2   # 间隔2秒

# 演奏唢呐曲目
./play.sh play molihua.json sn 100 30
sleep 30

./play.sh stop
```

### 场景3：定时播放（cron）
```bash
# 编辑crontab
crontab -e

# 添加定时任务：每天早上8点播放茉莉花
0 8 * * * cd /home/linkerhand/sks/sksgo && ./play.sh play molihua.json sn 100 30

# 每天早上8点10分停止
10 8 * * * cd /home/linkerhand/sks/sksgo && ./play.sh stop
```

### 场景4：紧急停止
如果演奏出现问题，可以随时执行：
```bash
./play.sh stop
```
无论当前状态如何，都会确保气泵关闭和设备复位。

## 参数详解

### 乐器类型
- `sks`：萨克斯
- `sn`：唢呐（也适用于葫芦丝、笛子等类似乐器）

### BPM（每分钟节拍数）
- `0`：使用音乐文件中的默认BPM
- `30-300`：自定义BPM值
- 推荐范围：60-180

### 吐音延迟
- 单位：毫秒（ms）
- 推荐范围：10-100ms
- 默认值：30ms
- 说明：控制相同音符之间的断气时间

## 错误处理

### 错误1：连接失败
```
curl: (7) Failed to connect to localhost port 8088
```
**原因：** Web服务未启动

**解决：**
```bash
# 检查服务是否运行
ps aux | grep sksgo

# 如果未运行，启动服务
go run main.go
```

### 错误2：文件不存在
```json
{
  "error": "音乐文件不存在"
}
```
**原因：** 指定的音乐文件不存在

**解决：**
```bash
# 检查文件是否存在
ls trsmusic/test.json

# 或列出所有音乐文件
ls trsmusic/*.json
```

### 错误3：演奏冲突
```json
{
  "error": "演奏正在进行中，请先停止当前演奏"
}
```
**原因：** 已有演奏正在进行

**解决：**
```bash
# 先停止当前演奏
./play.sh stop

# 然后重新开始
./play.sh play test.json
```

## 直接使用curl（不依赖脚本）

如果你更喜欢直接使用curl命令：

### 开始演奏
```bash
curl -X POST http://localhost:8088/api/playback/start \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "test.json",
    "instrument": "sks",
    "bpm": 120,
    "tonguing_delay": 30
  }'
```

### 停止演奏
```bash
curl -X POST http://localhost:8088/api/playback/stop \
  -H "Content-Type: application/json"
```

## Web界面改进

现在Web界面也修复了一个bug：
- **问题：** 之前点击停止后，需要重新选择文件才能再次演奏
- **修复：** 现在点击停止后，可以直接点击开始按钮重新演奏同一首曲目

## 注意事项

1. **音乐文件路径**
   - 所有音乐文件必须放在 `trsmusic/` 目录下
   - 脚本中只需要提供文件名，不需要路径

2. **参数顺序**
   - 参数必须按顺序提供
   - 如果要指定后面的参数，前面的参数不能省略

3. **并发控制**
   - 同一时间只能有一个演奏任务
   - 如果需要切换曲目，先停止再开始

4. **设备安全**
   - 停止命令会确保气泵关闭
   - 即使程序异常也不会损坏设备

## 总结

这个简化的脚本提供了最核心的两个功能：
- ✅ **play**：驱动8088端口开始演奏
- ✅ **stop**：驱动8088端口停止演奏

所有复杂的逻辑都由8088端口的Web服务处理，脚本只负责发送命令。


