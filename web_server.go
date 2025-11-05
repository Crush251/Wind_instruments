package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed web/static web/templates
var staticFiles embed.FS

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
	r.POST("/api/playback/stop", ws.stopPlayback)
	r.GET("/api/playback/status", ws.getPlaybackStatus)
	r.GET("/api/fingerings", ws.getFingeringMap)
	r.POST("/api/fingerings/send", ws.sendSingleFingering)

	// 预处理相关API
	r.POST("/api/preprocess", ws.preprocessSequence)
	r.GET("/api/exec/check", ws.checkExecFile)
	r.POST("/api/exec/play", ws.playExecSequence)

	// 气泵调试API
	r.POST("/api/pump/debug", ws.debugPumpCommand)

	// 静态文件服务（使用嵌入的文件系统）
	staticFS, _ := fs.Sub(staticFiles, "web/static")
	r.StaticFS("/static", http.FS(staticFS))

	// 模板加载（使用嵌入的文件系统）
	templatesFS, _ := fs.Sub(staticFiles, "web/templates")
	r.SetHTMLTemplate(ws.loadTemplates(templatesFS))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	fmt.Println("🎵 萨克斯/唢呐演奏Web服务启动成功!")
	fmt.Println("🌐 访问地址: http://localhost:1105")

	// 启动服务器
	if err := r.Run(":1105"); err != nil {
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

// StopPlayback 停止演奏（同步等待版本，确保完全停止）
func (ws *WebServer) stopPlayback(c *gin.Context) {
	fmt.Println("🛑 === 开始停止流程 ===")

	playbackController.mutex.RLock()
	isRunning := playbackController.isRunning
	instrument := playbackController.instrument
	cfg := playbackController.config
	playbackController.mutex.RUnlock()

	fmt.Printf("🔍 当前播放状态: isRunning=%v, instrument=%s\n", isRunning, instrument)

	if !isRunning {
		fmt.Println("ℹ️  没有正在运行的播放任务")
		c.JSON(http.StatusOK, gin.H{"message": "演奏已停止"})
		return
	}

	// 1. 立即关闭气泵（最优先）
	if globalPumpController != nil {
		fmt.Println("🔴 步骤1: 立即关闭气泵（使用同步方式）...")
		result := GlobalPumpOffSync()
		fmt.Printf("✅ 气泵关闭命令已执行，响应: %s\n", result)
	} else {
		fmt.Println("⚠️  气泵控制器为nil（可能是串口未连接）")
	}

	// 2. 发送停止信号并等待播放goroutine真正结束
	fmt.Println("📤 步骤2: 发送停止信号并等待播放完全停止...")
	select {
	case playbackController.stopChan <- true:
		fmt.Println("✅ 停止信号已发送")
	default:
		fmt.Println("⚠️  停止信号通道已满")
	}

	// 等待播放goroutine真正结束（最多等待3秒）
	fmt.Println("⏳ 等待播放goroutine完全退出...")
	select {
	case <-playbackController.doneChan:
		fmt.Println("✅ 播放goroutine已完全退出")
	case <-time.After(3 * time.Second):
		fmt.Println("⚠️  等待超时（3秒），强制继续")
	}

	// 3. 执行预备手势（松开手指）
	if instrument != "" {
		fmt.Printf("🤲 步骤3: 执行预备手势（松开手指，乐器: %s）...\n", instrument)
		readyController := NewReadyGestureController()
		readyController.ExecuteReadyGesture(cfg, instrument)
		fmt.Println("✅ 预备手势执行完成")
	} else {
		fmt.Println("⚠️  乐器类型为空，无法执行预备手势")
	}

	// 4. 更新状态
	playbackController.mutex.Lock()
	playbackController.isRunning = false
	playbackController.status.IsPlaying = false
	playbackController.mutex.Unlock()

	fmt.Println("✅ === 停止流程完成，可以安全启动新播放 ===")
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
		instrument = "sn" // 默认唢呐
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
		Instrument string `json:"instrument"` // "sn" 或 "sks"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 默认乐器类型
	if request.Instrument == "" {
		request.Instrument = "sn"
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

////////////////////////////////////////////////////////////////////////////////
// 预处理相关API
////////////////////////////////////////////////////////////////////////////////

// preprocessSequence 预处理音乐文件生成执行序列
func (ws *WebServer) preprocessSequence(c *gin.Context) {
	var request struct {
		SourceFile    string  `json:"source_file"`
		Instrument    string  `json:"instrument"`
		BPM           float64 `json:"bpm"`
		TonguingDelay int     `json:"tonguing_delay"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 确保exec目录存在
	execDir := "exec"
	if err := os.MkdirAll(execDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建exec目录失败: %v", err)})
		return
	}

	// 生成输出文件名
	baseFilename := filepath.Base(request.SourceFile)
	baseFilename = baseFilename[:len(baseFilename)-5] // 移除.json
	outputFilename := fmt.Sprintf("%s_%s_%.0f_%d.exec.json",
		baseFilename, request.Instrument, request.BPM, request.TonguingDelay)
	outputPath := filepath.Join(execDir, outputFilename)

	// 加载配置和指法映射
	cfg := ws.fileReader.LoadConfig("config.yaml")
	fingeringMap := ws.fileReader.LoadFingeringMapByInstrument(request.Instrument)

	// 获取BPM
	bpm := request.BPM
	if bpm <= 0 {
		bpm = cfg.BPM
		if bpm <= 0 {
			bpm = 60
		}
	}

	// 创建预处理器
	preprocessor := NewSequencePreprocessor(cfg, fingeringMap, request.Instrument, bpm, request.TonguingDelay)

	// 生成执行序列
	if err := preprocessor.GenerateExecutionSequence(request.SourceFile, outputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("预处理失败: %v", err)})
		return
	}

	// 读取生成的序列文件获取元数据
	sequence, err := loadExecutionSequence(outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("读取序列文件失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "预处理完成",
		"exec_file":    outputFilename,
		"exec_path":    outputPath,
		"total_events": sequence.Meta.TotalEvents,
		"duration_ms":  sequence.Meta.TotalDurationMS,
		"duration_sec": sequence.Meta.TotalDurationMS / 1000.0,
	})
}

// checkExecFile 检查执行序列文件是否存在
func (ws *WebServer) checkExecFile(c *gin.Context) {
	sourceFile := c.Query("source_file")
	instrument := c.Query("instrument")
	bpm := c.Query("bpm")
	tonguingDelay := c.Query("tonguing_delay")

	if sourceFile == "" || instrument == "" || bpm == "" || tonguingDelay == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	// 生成预期的文件名
	baseFilename := filepath.Base(sourceFile)
	baseFilename = baseFilename[:len(baseFilename)-5]
	execFilename := fmt.Sprintf("%s_%s_%s_%s.exec.json",
		baseFilename, instrument, bpm, tonguingDelay)
	execPath := filepath.Join("exec", execFilename)

	// 检查文件是否存在
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"exists":    false,
			"exec_file": execFilename,
		})
		return
	}

	// 读取序列文件获取元数据
	sequence, err := loadExecutionSequence(execPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
			"error":  fmt.Sprintf("文件损坏: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":       true,
		"exec_file":    execFilename,
		"exec_path":    execPath,
		"total_events": sequence.Meta.TotalEvents,
		"duration_ms":  sequence.Meta.TotalDurationMS,
		"duration_sec": sequence.Meta.TotalDurationMS / 1000.0,
	})
}

// playExecSequence 播放预计算的执行序列
func (ws *WebServer) playExecSequence(c *gin.Context) {
	var request struct {
		ExecFile string `json:"exec_file"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 构建完整路径
	execPath := filepath.Join("exec", request.ExecFile)

	// 检查文件是否存在
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行序列文件不存在"})
		return
	}

	// 停止当前播放（如果有）
	if playbackController.isRunning {
		fmt.Println("⚠️  检测到正在播放，先停止旧的播放任务...")
		select {
		case playbackController.stopChan <- true:
			fmt.Println("✅ 停止信号已发送")
		default:
			fmt.Println("⚠️  停止信号通道已满")
		}

		// 等待旧播放完全停止
		fmt.Println("⏳ 等待旧播放完全停止...")
		select {
		case <-playbackController.doneChan:
			fmt.Println("✅ 旧播放已完全停止")
		case <-time.After(2 * time.Second):
			fmt.Println("⚠️  等待超时（2秒），强制继续")
		}

		// 短暂延迟确保资源释放
		time.Sleep(100 * time.Millisecond)
	}

	// 加载配置
	cfg := ws.fileReader.LoadConfig("config.yaml")

	// 创建执行引擎
	engine, err := NewExecutionEngine(execPath, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建执行引擎失败: %v", err)})
		return
	}

	// 异步开始播放
	if err := engine.PlayAsync(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("启动播放失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "开始播放执行序列",
		"exec_file":    request.ExecFile,
		"total_events": engine.sequence.Meta.TotalEvents,
		"duration_sec": engine.sequence.Meta.TotalDurationMS / 1000.0,
	})
}

// loadTemplates 加载嵌入的模板文件
func (ws *WebServer) loadTemplates(templatesFS fs.FS) *template.Template {
	tmpl := template.New("")

	fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := fs.ReadFile(templatesFS, path)
		if err != nil {
			return err
		}

		_, err = tmpl.New(filepath.Base(path)).Parse(string(content))
		return err
	})

	return tmpl
}

// debugPumpCommand 处理气泵调试命令
func (ws *WebServer) debugPumpCommand(c *gin.Context) {
	var request struct {
		Command string `json:"command"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	if request.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "命令不能为空"})
		return
	}

	// 检查气泵控制器是否已初始化
	if globalPumpController == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "气泵控制器未初始化"})
		return
	}

	// 发送命令到串口
	fmt.Printf("🔧 调试命令: %s\n", request.Command)

	// 使用全局气泵控制器发送命令（同步版本，等待响应）
	response := GlobalPumpSendSync(request.Command)

	// 检查响应
	if response == "气泵控制器未初始化" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "气泵控制器未初始化",
			"details": response,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "命令发送成功",
		"command":  request.Command,
		"response": response,
	})
}
