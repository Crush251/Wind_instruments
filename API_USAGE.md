# API 使用指南

本文档介绍如何通过命令行脚本或编程方式控制萨克斯/唢呐演奏系统。

## 目录

- [前提条件](#前提条件)
- [Web页面中断处理改进](#web页面中断处理改进)
- [使用Shell脚本控制](#使用shell脚本控制)
- [使用Python脚本控制](#使用python脚本控制)
- [使用curl命令](#使用curl命令)
- [API端点说明](#api端点说明)

---

## 前提条件

1. **启动Web服务**
   ```bash
   cd /home/linkerhand/sks/sksgo
   go run main.go
   # 或者使用编译后的可执行文件
   ./sksgo
   ```
   
   服务将在 `http://localhost:8088` 上运行。

2. **检查服务是否正常**
   ```bash
   curl http://localhost:8088/api/playback/status
   ```

---

## Web页面中断处理改进

### 改进内容

我们对Web前端的中断逻辑进行了以下改进：

1. **页面刷新时自动停止演奏**
   - 使用 `beforeunload` 事件监听页面刷新
   - 使用 `navigator.sendBeacon()` 确保停止请求能发出

2. **页面关闭时自动停止演奏**
   - 监听 `pagehide` 事件
   - 确保浏览器标签页关闭时演奏停止

3. **后端容错处理**
   - 即使没有演奏在进行，停止请求也会确保气泵关闭和手势复位
   - 避免状态不一致导致的问题

### 技术实现

**前端（web/static/js/app.js）：**
```javascript
// 页面卸载时停止演奏
window.addEventListener('beforeunload', function(e) {
    if (isPlaying) {
        // 使用sendBeacon确保请求能发出去（即使页面正在关闭）
        navigator.sendBeacon('/api/playback/stop', JSON.stringify({}));
    }
});

// 页面隐藏时的处理
window.addEventListener('pagehide', function() {
    if (isPlaying) {
        navigator.sendBeacon('/api/playback/stop', JSON.stringify({}));
    }
});
```

**后端（web_server.go）：**
```go
// 停止请求现在会确保气泵关闭，即使没有演奏在进行
if !isRunning {
    utils.ControlAirPumpWithLock(playbackController.config, false)
    readyController.ExecuteReadyGesture(playbackController.config, playbackController.instrument)
}
```

---

## 使用Shell脚本控制

### 基本用法

```bash
# 交互式菜单模式
./api_examples.sh

# 直接执行命令
./api_examples.sh <命令> [参数...]
```

### 常用命令示例

#### 1. 列出所有音乐文件
```bash
./api_examples.sh list
```

#### 2. 开始演奏萨克斯
```bash
# 使用默认BPM
./api_examples.sh play-sax test.json

# 快速演奏（会显示演奏状态）
./api_examples.sh quick-sax test.json
```

#### 3. 开始演奏唢呐（自定义BPM和吐音延迟）
```bash
# 参数：文件名 BPM 吐音延迟(ms)
./api_examples.sh play-suona molihua.json 120 30

# 快速演奏
./api_examples.sh quick-suona molihua.json 120 30
```

#### 4. 停止演奏
```bash
./api_examples.sh stop

# 或使用快捷命令
./api_examples.sh quick-stop
```

#### 5. 查看演奏状态
```bash
./api_examples.sh status
```

#### 6. 暂停/恢复演奏
```bash
./api_examples.sh pause
```

#### 7. 获取指法映射
```bash
# 萨克斯指法
./api_examples.sh fingerings-sax

# 唢呐指法
./api_examples.sh fingerings-suona
```

#### 8. 发送单个指法
```bash
# 参数：音符 乐器类型
./api_examples.sh send-note A4 sks
./api_examples.sh send-note C5 sn
```

#### 9. 获取歌曲时间轴
```bash
./api_examples.sh timeline test.json
```

### 实际使用场景

**场景1：快速测试某个音乐文件**
```bash
# 开始演奏
./api_examples.sh quick-sax test.json

# 等待一段时间后停止
sleep 10
./api_examples.sh quick-stop
```

**场景2：在脚本中批量测试多个文件**
```bash
#!/bin/bash
for file in test.json molihua.json waitansks.json; do
    echo "正在测试: $file"
    ./api_examples.sh play-sax "$file"
    sleep 5  # 播放5秒
    ./api_examples.sh stop
    sleep 2  # 间隔2秒
done
```

**场景3：在cron定时任务中使用**
```bash
# 每天早上8点播放茉莉花（唢呐，BPM 100）
0 8 * * * cd /home/linkerhand/sks/sksgo && ./api_examples.sh play-suona molihua.json 100 30
```

---

## 使用Python脚本控制

### 安装依赖

```bash
pip3 install requests
```

### 基本用法

```bash
python3 api_examples.py <命令> [参数...]
```

### 常用命令示例

#### 1. 列出音乐文件
```bash
python3 api_examples.py list

# 搜索特定文件
python3 api_examples.py list molihua
```

#### 2. 开始演奏
```bash
# 萨克斯（使用默认BPM）
python3 api_examples.py play test.json

# 萨克斯（指定BPM和吐音延迟）
python3 api_examples.py play test.json 120 30

# 唢呐
python3 api_examples.py play-suona molihua.json 100 30
```

#### 3. 停止演奏
```bash
python3 api_examples.py stop
```

#### 4. 查看状态
```bash
python3 api_examples.py status
```

#### 5. 获取指法
```bash
# 萨克斯指法
python3 api_examples.py fingerings sks

# 唢呐指法
python3 api_examples.py fingerings sn
```

#### 6. 发送单个指法
```bash
python3 api_examples.py send-note A4 sks
```

#### 7. 获取时间轴
```bash
python3 api_examples.py timeline test.json
```

### 在Python程序中使用

```python
from api_examples import MusicController

# 创建控制器
controller = MusicController("http://localhost:8088/api")

# 列出音乐文件
files = controller.get_music_files()
print(f"共有 {files['total']} 个音乐文件")

# 开始演奏
result = controller.start_playback(
    filename="test.json",
    instrument="sks",  # sks=萨克斯, sn=唢呐
    bpm=120,           # 0表示使用默认BPM
    tonguing_delay=30  # 吐音延迟（毫秒）
)
print(result['message'])

# 获取状态
status = controller.get_playback_status()
print(f"当前演奏: {status['current_file']}")
print(f"进度: {status['progress']}%")

# 停止演奏
controller.stop_playback()
```

---

## 使用curl命令

### 1. 获取音乐文件列表

```bash
curl -s http://localhost:8088/api/files | jq '.'
```

### 2. 开始演奏（萨克斯）

```bash
curl -X POST http://localhost:8088/api/playback/start \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "test.json",
    "instrument": "sks",
    "bpm": 0,
    "tonguing_delay": 30
  }' | jq '.'
```

### 3. 开始演奏（唢呐，自定义BPM）

```bash
curl -X POST http://localhost:8088/api/playback/start \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "molihua.json",
    "instrument": "sn",
    "bpm": 120,
    "tonguing_delay": 30
  }' | jq '.'
```

### 4. 停止演奏

```bash
curl -X POST http://localhost:8088/api/playback/stop \
  -H "Content-Type: application/json" | jq '.'
```

### 5. 暂停/恢复演奏

```bash
curl -X POST http://localhost:8088/api/playback/pause \
  -H "Content-Type: application/json" | jq '.'
```

### 6. 获取演奏状态

```bash
curl -s http://localhost:8088/api/playback/status | jq '.'
```

### 7. 获取指法映射

```bash
# 萨克斯
curl -s "http://localhost:8088/api/fingerings?instrument=sks" | jq '.'

# 唢呐
curl -s "http://localhost:8088/api/fingerings?instrument=sn" | jq '.'
```

### 8. 发送单个指法

```bash
curl -X POST http://localhost:8088/api/fingerings/send \
  -H "Content-Type: application/json" \
  -d '{
    "note": "A4",
    "instrument": "sks"
  }' | jq '.'
```

### 9. 获取歌曲时间轴

```bash
curl -s "http://localhost:8088/api/timeline?filename=test.json" | jq '.'
```

### 10. 更新歌曲时间轴（修改空拍时长）

```bash
curl -X POST http://localhost:8088/api/timeline/update \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "test.json",
    "timeline": [
      ["A4", 1],
      ["B4", 0.5],
      ["NO", 2],
      ["C5", 1]
    ]
  }' | jq '.'
```

---

## API端点说明

### 1. GET /api/files
获取音乐文件列表

**查询参数：**
- `search` (可选): 搜索关键词

**响应示例：**
```json
{
  "files": [
    {
      "filename": "test.json",
      "title": "测试曲目",
      "bpm": 120,
      "duration": 50,
      "file_path": "trsmusic/test.json",
      "file_size": 1024,
      "modified_at": "2025-10-17 10:30:00"
    }
  ],
  "total": 1
}
```

### 2. POST /api/playback/start
开始演奏

**请求体：**
```json
{
  "filename": "test.json",
  "instrument": "sks",
  "bpm": 120,
  "tonguing_delay": 30
}
```

**参数说明：**
- `filename`: 音乐文件名（必需）
- `instrument`: 乐器类型，`sks`=萨克斯，`sn`=唢呐（可选，默认`sks`）
- `bpm`: 节拍速度，`0`表示使用文件默认值（可选，默认`0`）
- `tonguing_delay`: 吐音延迟（毫秒）（可选，默认`30`）

**响应示例：**
```json
{
  "message": "演奏已开始"
}
```

### 3. POST /api/playback/pause
暂停/恢复演奏

**响应示例：**
```json
{
  "message": "演奏已暂停"
}
```

### 4. POST /api/playback/stop
停止演奏

**响应示例：**
```json
{
  "message": "演奏已停止"
}
```

### 5. GET /api/playback/status
获取演奏状态

**响应示例：**
```json
{
  "is_playing": true,
  "is_paused": false,
  "current_file": "test.json",
  "current_note": 10,
  "total_notes": 50,
  "elapsed_time": "5s",
  "remaining_time": "20s",
  "progress": 20.0
}
```

### 6. GET /api/fingerings
获取指法映射

**查询参数：**
- `instrument`: 乐器类型（`sks` 或 `sn`）

**响应示例：**
```json
{
  "fingerings": [
    {
      "note": "A4",
      "left": ["Thumb", "Index", "Middle"],
      "right": []
    }
  ]
}
```

### 7. POST /api/fingerings/send
发送单个指法

**请求体：**
```json
{
  "note": "A4",
  "instrument": "sks"
}
```

**响应示例：**
```json
{
  "message": "已发送音符 A4 的指法"
}
```

### 8. GET /api/timeline
获取歌曲时间轴

**查询参数：**
- `filename`: 音乐文件名

**响应示例：**
```json
{
  "filename": "test.json",
  "bpm": 120,
  "timeline": [
    ["A4", 1],
    ["B4", 0.5],
    ["NO", 2]
  ],
  "meta": {
    "title": "测试曲目",
    "bpm": 120
  }
}
```

### 9. POST /api/timeline/update
更新歌曲时间轴

**请求体：**
```json
{
  "filename": "test.json",
  "timeline": [
    ["A4", 1],
    ["B4", 0.5],
    ["NO", 2]
  ]
}
```

**响应示例：**
```json
{
  "message": "时间轴更新成功",
  "filename": "test.json"
}
```

---

## 注意事项

1. **确保Web服务已启动**
   - 所有API调用都需要Web服务运行在 `http://localhost:8088`

2. **音乐文件路径**
   - 所有音乐文件应放在 `trsmusic/` 目录下
   - API中只需提供文件名（如 `test.json`），不需要完整路径

3. **中断处理**
   - Web页面刷新或关闭时会自动停止演奏
   - 程序退出（Ctrl+C）时会自动关闭气泵
   - 命令行脚本可随时调用停止API

4. **并发控制**
   - 同一时间只能有一个演奏任务
   - 如果尝试在演奏进行中开始新演奏，会返回冲突错误

5. **气泵和指法控制**
   - 停止演奏时会自动关闭气泵并复位到预备手势
   - 即使没有演奏在进行，调用停止API也会确保设备处于安全状态

---

## 故障排除

### 1. API请求失败

**问题：** `Connection refused` 或 `Failed to connect`

**解决方案：**
```bash
# 检查Web服务是否运行
ps aux | grep sksgo

# 如果没有运行，启动服务
cd /home/linkerhand/sks/sksgo
go run main.go
```

### 2. 停止命令无效

**问题：** 调用停止API后演奏仍在继续

**解决方案：**
```bash
# 多次调用停止命令
./api_examples.sh stop
sleep 1
./api_examples.sh stop

# 或直接重启程序
pkill -f "go run main.go"
go run main.go
```

### 3. jq命令未找到

**问题：** Shell脚本提示 `jq: command not found`

**解决方案：**
```bash
# Ubuntu/Debian
sudo apt-get install jq

# 或者移除脚本中的 | jq '.' 部分
```

---

## 更多示例

查看以下文件获取更多代码示例：
- `api_examples.sh` - Shell脚本示例
- `api_examples.py` - Python脚本示例
- `web/static/js/app.js` - 前端JavaScript实现


