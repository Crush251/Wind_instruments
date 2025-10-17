package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

////////////////////////////////////////////////////////////////////////////////
// Web服务模块
////////////////////////////////////////////////////////////////////////////////

// WebServer Web服务器
type WebServer struct {
	fileReader   *FileReader
	musicScanner *MusicFileScanner
}

// NewWebServer 创建新的Web服务器
func NewWebServer() *WebServer {
	return &WebServer{
		fileReader:   NewFileReader(),
		musicScanner: NewMusicFileScanner(),
	}
}

// StartWebServer 启动Web服务器
func (ws *WebServer) StartWebServer() {
	// 设置Gin为发布模式（减少日志输出）
	gin.SetMode(gin.ReleaseMode)

	// 创建轻量级路由（不使用默认的Logger和Recovery中间件）
	r := gin.New()

	// 只添加必要的中间件
	r.Use(gin.Recovery()) // 只保留错误恢复，移除详细日志

	// 允许跨域
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API路由
	r.GET("/api/files", ws.getMusicFiles)
	r.GET("/api/timeline", ws.getTimeline)
	r.POST("/api/timeline/update", ws.updateTimeline)
	r.POST("/api/playback/start", ws.startPlayback)
	r.POST("/api/playback/pause", ws.pausePlayback)
	r.POST("/api/playback/stop", ws.stopPlayback)
	r.GET("/api/playback/status", ws.getPlaybackStatus)
	r.GET("/api/fingerings", ws.getFingeringMap)
	r.POST("/api/fingerings/send", ws.sendSingleFingering)

	// 静态文件服务（前端）
	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("web/templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	fmt.Println("🎵 萨克斯/唢呐演奏Web服务启动成功!")
	fmt.Println("🌐 访问地址: http://localhost:8088")

	// 启动服务器
	if err := r.Run(":8088"); err != nil {
		fmt.Printf("❌ Web服务启动失败: %v\n", err)
	}
}

// GetTimeline 获取歌曲时间轴数据
func (ws *WebServer) getTimeline(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少filename参数"})
		return
	}

	// 加载时间轴文件
	fpath := filepath.Join("trsmusic", filename)
	if err := ws.fileReader.CheckFileExists(fpath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐文件不存在"})
		return
	}

	timeline := ws.fileReader.LoadTimeline(fpath)

	// 提取BPM
	bpm := 60.0
	if bpmVal, exists := timeline.Meta["bpm"]; exists {
		utils := NewUtils()
		if bpmFloat, ok := utils.ConvertToFloat(bpmVal); ok && bpmFloat > 0 {
			bpm = bpmFloat
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"bpm":      bpm,
		"timeline": timeline.Timeline,
		"meta":     timeline.Meta,
	})
}

// UpdateTimeline 更新时间轴数据（保存到JSON文件）
func (ws *WebServer) updateTimeline(c *gin.Context) {
	var request struct {
		Filename string        `json:"filename"`
		Timeline []interface{} `json:"timeline"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 构造文件路径
	fpath := filepath.Join("trsmusic", request.Filename)

	// 检查文件是否存在
	if err := ws.fileReader.CheckFileExists(fpath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐文件不存在"})
		return
	}

	// 读取原始文件
	data, err := os.ReadFile(fpath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 解析JSON
	var fileData map[string]interface{}
	if err := json.Unmarshal(data, &fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析文件失败"})
		return
	}

	// 更新timeline字段
	fileData["timeline"] = request.Timeline

	// 写回文件（格式化JSON）
	newData, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成JSON失败"})
		return
	}

	if err := os.WriteFile(fpath, newData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "时间轴更新成功",
		"filename": request.Filename,
	})
}

// GetMusicFiles 获取音乐文件列表
func (ws *WebServer) getMusicFiles(c *gin.Context) {
	search := c.Query("search") // 搜索关键词

	files, err := ws.musicScanner.GetMusicFileList("trsmusic", search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("扫描音乐文件失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"total": len(files),
	})
}

// StartPlayback 开始演奏
func (ws *WebServer) startPlayback(c *gin.Context) {
	var request struct {
		Filename      string  `json:"filename"`
		Instrument    string  `json:"instrument"`     // "sks" 或 "sn"
		BPM           float64 `json:"bpm"`            // 用户指定的BPM，0表示使用默认
		TonguingDelay int     `json:"tonguing_delay"` // 吐音延迟时间（毫秒）
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 默认乐器类型
	if request.Instrument == "" {
		request.Instrument = "sks"
	}

	// 默认吐音延迟
	if request.TonguingDelay <= 0 {
		request.TonguingDelay = 30
	}

	// 检查是否已在演奏
	playbackController.mutex.RLock()
	isRunning := playbackController.isRunning
	playbackController.mutex.RUnlock()

	if isRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "演奏正在进行中，请先停止当前演奏"})
		return
	}

	// 加载音乐文件
	fpath := filepath.Join("trsmusic", request.Filename)
	if err := ws.fileReader.CheckFileExists(fpath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐文件不存在"})
		return
	}

	// 启动演奏
	go func() {
		startPerformanceAsyncWithParams(fpath, request.Instrument, request.BPM, request.TonguingDelay, ws.fileReader)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "演奏已开始"})
}

// PausePlayback 暂停/恢复演奏
func (ws *WebServer) pausePlayback(c *gin.Context) {
	playbackController.mutex.RLock()
	isRunning := playbackController.isRunning
	isPaused := playbackController.status.IsPaused
	playbackController.mutex.RUnlock()

	if !isRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前没有演奏在进行"})
		return
	}

	if isPaused {
		// 恢复演奏
		select {
		case playbackController.resumeChan <- true:
		default:
		}
		c.JSON(http.StatusOK, gin.H{"message": "演奏已恢复"})
	} else {
		// 暂停演奏
		select {
		case playbackController.pauseChan <- true:
		default:
		}
		c.JSON(http.StatusOK, gin.H{"message": "演奏已暂停"})
	}
}

// StopPlayback 停止演奏
func (ws *WebServer) stopPlayback(c *gin.Context) {
	playbackController.mutex.RLock()
	isRunning := playbackController.isRunning
	playbackController.mutex.RUnlock()

	if !isRunning {
		// 即使没有演奏在进行，也确保气泵关闭和手势复位
		utils := NewUtils()
		if playbackController.config.CanBridgeURL != "" {
			utils.ControlAirPumpWithLock(playbackController.config, false)
			readyController := NewReadyGestureController()
			if playbackController.instrument != "" && playbackController.config.Ready.Enabled {
				readyController.ExecuteReadyGesture(playbackController.config, playbackController.instrument)
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "演奏已停止（或未在进行）"})
		return
	}

	// 发送停止信号
	select {
	case playbackController.stopChan <- true:
	default:
	}

	// 停止演奏恢复到预演奏手势
	utils := NewUtils()
	utils.ControlAirPumpWithLock(playbackController.config, false)
	readyController := NewReadyGestureController()
	readyController.ExecuteReadyGesture(playbackController.config, playbackController.instrument)

	c.JSON(http.StatusOK, gin.H{"message": "演奏已停止"})
}

// GetPlaybackStatus 获取演奏状态
func (ws *WebServer) getPlaybackStatus(c *gin.Context) {
	playbackController.mutex.RLock()
	status := playbackController.status
	playbackController.mutex.RUnlock()

	c.JSON(http.StatusOK, status)
}

// GetFingeringMap 获取指法映射
func (ws *WebServer) getFingeringMap(c *gin.Context) {
	instrument := c.Query("instrument") // 获取乐器类型参数
	if instrument == "" {
		instrument = "sks" // 默认萨克斯
	}

	fingeringMap := ws.fileReader.LoadFingeringMapByInstrument(instrument)

	// 转换为前端友好的格式
	var fingerings []gin.H
	for note, entry := range fingeringMap {
		fingerings = append(fingerings, gin.H{
			"note":  note,
			"left":  entry.Left,
			"right": entry.Right,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"fingerings": fingerings,
	})
}

// SendSingleFingering 发送单个指法
func (ws *WebServer) sendSingleFingering(c *gin.Context) {
	var request struct {
		Note       string `json:"note"`
		Instrument string `json:"instrument"` // "sks" 或 "sn"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 默认乐器类型
	if request.Instrument == "" {
		request.Instrument = "sks"
	}

	// 加载配置和指法映射
	cfg := ws.fileReader.LoadConfig("config.yaml")
	fingeringMap := ws.fileReader.LoadFingeringMapByInstrument(request.Instrument)

	fingering, exists := fingeringMap[request.Note]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("未找到音符 %s 的指法映射", request.Note)})
		return
	}

	// 发送指法
	utils := NewUtils()
	if err := utils.SwitchFingeringWithLogging(cfg, fingering, request.Instrument); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("发送指法失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("已发送音符 %s 的指法", request.Note)})
}
