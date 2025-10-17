package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial.v1"
)

////////////////////////////////////////////////////////////////////////////////
// 工具函数模块
////////////////////////////////////////////////////////////////////////////////

// 全局气泵控制器
var globalPumpController *PumpController

// 全局HTTP客户端（连接池复用，显著提升性能）
var globalHTTPClient *http.Client
var httpClientOnce sync.Once

// InitGlobalHTTPClient 初始化全局HTTP客户端（带连接池）
func InitGlobalHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		globalHTTPClient = &http.Client{
			Timeout: 100 * time.Millisecond, // 设置100ms超时，避免阻塞
			Transport: &http.Transport{
				MaxIdleConns:        100,              // 最大空闲连接数
				MaxIdleConnsPerHost: 10,               // 每个主机最大空闲连接
				IdleConnTimeout:     90 * time.Second, // 空闲连接超时
				DisableKeepAlives:   false,            // 启用Keep-Alive
				DisableCompression:  true,             // 禁用压缩以提高速度
			},
		}
	})
	return globalHTTPClient
}

// Utils 工具函数集合
type Utils struct{}

// PumpController 气泵控制器
type PumpController struct {
	port serial.Port
}

// NewUtils 创建新的工具函数实例
func NewUtils() *Utils {
	return &Utils{}
}

// InitGlobalPumpController 初始化全局气泵控制器
func InitGlobalPumpController(portName string) error {
	if globalPumpController != nil {
		// 如果已经初始化，先关闭
		CloseGlobalPumpController()
	}

	mode := &serial.Mode{BaudRate: 9600}
	port, err := serial.Open(portName, mode)
	if err != nil {
		// 如果端口检测不到，尝试 /dev/ttyUSB1
		altPorts := []string{"/dev/ttyUSB1", "/dev/ttyUSB2"}
		found := false
		for _, altPort := range altPorts {
			port, err = serial.Open(altPort, mode)
			if err == nil {
				fmt.Printf("⚠️  指定端口'%s'未连接，已切换到可用端口：%s\n", portName, altPort)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("无法打开串口: %s, 已尝试其他端口且失败，最后错误: %v", portName, err)
		}
	}

	//time.Sleep(1 * time.Second)
	port.ResetInputBuffer()

	globalPumpController = &PumpController{port: port}

	// 设置为手动模式并关闭气泵
	GlobalPumpManual()
	GlobalPumpOff()

	fmt.Printf("✅ 气泵控制器初始化成功，串口: %s\n", portName)
	return nil
}

// CloseGlobalPumpController 关闭全局气泵控制器
func CloseGlobalPumpController() {
	if globalPumpController != nil && globalPumpController.port != nil {
		// 确保气泵关闭
		GlobalPumpOff()
		globalPumpController.port.Close()
		globalPumpController = nil
		fmt.Println("✅ 气泵控制器已关闭")
	}
}

// NewPumpController 创建新的气泵控制器（已废弃，使用全局控制器）
func (u *Utils) NewPumpController(portName string) error {
	return InitGlobalPumpController(portName)
}

// ClosePumpController 关闭气泵控制器（已废弃，使用全局控制器）
func (u *Utils) ClosePumpController() {
	CloseGlobalPumpController()
}

// GlobalPumpSend 发送气泵命令（异步版本，不等待响应，提高演奏速度）
func GlobalPumpSend(cmd string) string {
	if globalPumpController == nil || globalPumpController.port == nil {
		return "气泵控制器未初始化"
	}

	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	globalPumpController.port.Write([]byte(cmd))

	// 演奏过程中不需要等待响应，立即返回以避免延迟累积
	return "OK"
}

// PumpSend 发送气泵命令（实例版本，已废弃）
func (u *Utils) PumpSend(cmd string) string {
	return GlobalPumpSend(cmd)
}

// GlobalPumpHelp 获取气泵帮助信息（全局版本）
func GlobalPumpHelp() string { return GlobalPumpSend("help") }

// GlobalPumpAuto 设置气泵为自动模式（全局版本）
func GlobalPumpAuto() string { return GlobalPumpSend("auto") }

// GlobalPumpManual 设置气泵为手动模式（全局版本）
func GlobalPumpManual() string { return GlobalPumpSend("manual") }

// GlobalPumpOn 开启气泵（全局版本）
func GlobalPumpOn() string { return GlobalPumpSend("on") }

// GlobalPumpOff 关闭气泵（全局版本）
func GlobalPumpOff() string { return GlobalPumpSend("off") }

// GlobalPumpSetPWM 设置气泵PWM值（全局版本）
func GlobalPumpSetPWM(value int) string {
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	return GlobalPumpSend(fmt.Sprintf("set %d", value))
}

// GlobalPumpSetSpeed 设置气泵变化速度（全局版本）
func GlobalPumpSetSpeed(step int) string {
	if step < 1 {
		step = 1
	} else if step > 50 {
		step = 50
	}
	return GlobalPumpSend(fmt.Sprintf("speed %d", step))
}

// GlobalPumpStatus 获取气泵状态（全局版本）
func GlobalPumpStatus() map[string]string {
	result := map[string]string{"raw": GlobalPumpSend("status")}
	lines := strings.Split(result["raw"], "\n")

	for _, line := range lines {
		switch {
		case strings.Contains(line, "模式"):
			if strings.Contains(line, "自动") {
				result["mode"] = "自动"
			} else {
				result["mode"] = "手动"
			}
		case strings.Contains(line, "当前PWM值"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				result["pwm_raw"] = strings.TrimSpace(parts[1])
			}
		case strings.Contains(line, "变化速度"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				result["speed"] = strings.TrimSpace(parts[1])
			}
		}
	}
	return result
}

// ParseCanID 解析CAN设备ID（支持十六进制和十进制）
func (u *Utils) ParseCanID(idStr string) uint32 {
	idStr = strings.TrimSpace(idStr)

	if strings.HasPrefix(idStr, "0x") || strings.HasPrefix(idStr, "0X") {
		// 十六进制格式
		if val, err := strconv.ParseUint(idStr[2:], 16, 32); err == nil {
			return uint32(val)
		}
	} else {
		// 十进制格式
		if val, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			return uint32(val)
		}
	}

	return 0x28 // 默认值
}

// ConvertToFloat 将任意数值类型转换为float64
func (u *Utils) ConvertToFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// SendCanFrame 发送CAN数据帧（同步版本）
func (u *Utils) SendCanFrame(cfg Config, iface string, id uint32, data []byte) error {
	msg := CanMessage{
		Interface: iface,
		Id:        id,
		Data:      data,
	}

	if cfg.DryRun {
		return nil
	}

	return u.ForwardToCanService(cfg.CanBridgeURL, msg)
}

// SendCanFrameAsync 异步发送CAN数据帧（不等待响应，极速模式）
// 适用于演奏过程中的高频指法切换
func (u *Utils) SendCanFrameAsync(cfg Config, iface string, id uint32, data []byte) {
	if cfg.DryRun {
		return
	}

	msg := CanMessage{
		Interface: iface,
		Id:        id,
		Data:      data,
	}

	u.ForwardToCanServiceAsync(cfg.CanBridgeURL, msg)
}

// ForwardToCanService 转发消息到CAN桥接服务（同步版本，等待响应）
func (u *Utils) ForwardToCanService(canBridgeURL string, msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败: %v", err)
	}

	// 使用全局HTTP客户端（连接池复用）
	client := InitGlobalHTTPClient()
	resp, err := client.Post(canBridgeURL+"/api/can", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("发送到CAN服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CAN服务错误: %s", string(body))
	}

	return nil
}

// ForwardToCanServiceAsync 异步转发消息到CAN桥接服务（不等待响应，极速模式）
// 适用于演奏过程中的高频指法切换，显著降低延迟
func (u *Utils) ForwardToCanServiceAsync(canBridgeURL string, msg CanMessage) {
	go func() {
		jsonData, err := json.Marshal(msg)
		if err != nil {
			// 异步模式下，错误不影响主流程，仅记录（可选）
			return
		}

		// 使用全局HTTP客户端（连接池复用）
		client := InitGlobalHTTPClient()
		resp, err := client.Post(canBridgeURL+"/api/can", "application/json", bytes.NewReader(jsonData))
		if err != nil {
			// 异步模式下，错误不影响主流程
			return
		}
		defer resp.Body.Close()

		// 不检查响应状态，快速返回
		if resp.StatusCode != http.StatusOK {
			// 可选：记录错误，但不阻塞演奏
			io.ReadAll(resp.Body) // 读取body以释放连接
		}
	}()
}

// ControlAirPumpWithLock 控制气泵开关（同步版本，等待操作完成）
// 参数：cfg - 配置信息，用于判断使用串口还是CAN通信
// on - true为开启，false为关闭
func (u *Utils) ControlAirPumpWithLock(cfg Config, on bool) error {
	// 根据配置选择通信方式
	if cfg.Pump.UseSerial && cfg.Pump.PortName != "" {
		// 使用串口通信
		// 检查全局气泵控制器是否已初始化
		if globalPumpController == nil {
			return fmt.Errorf("气泵控制器未初始化，请先调用InitGlobalPumpController")
		}

		if on {
			// 开启气泵
			GlobalPumpOn()
		} else {
			// 关闭气泵
			GlobalPumpOff()
		}

		return nil
	}

	// 使用CAN通信方式
	return u.ControlAirPumpWithCAN(on)
}

// ControlAirPumpAsync 异步控制气泵开关（不等待完成，极速模式）
// 适用于演奏过程中的高频气泵开关，主程序严格按BPM时间推进
func (u *Utils) ControlAirPumpAsync(cfg Config, on bool) {
	go func() {
		// 根据配置选择通信方式
		if cfg.Pump.UseSerial && cfg.Pump.PortName != "" {
			// 使用串口通信
			if globalPumpController == nil {
				return // 异步模式下，错误不影响主流程
			}

			if on {
				GlobalPumpOn()
			} else {
				GlobalPumpOff()
			}
		} else {
			// 使用CAN通信方式
			u.ControlAirPumpWithCANAsync(on)
		}
	}()
}

// ControlAirPumpWithCAN 使用CAN通信控制气泵（同步版本）
func (u *Utils) ControlAirPumpWithCAN(on bool) error {
	QBmsg := CanMessage{
		Interface: "can4", // 默认气泵接口
		Id:        0x101,
		Data:      []byte{},
	}

	if on {
		// 发送开启气泵的指令
		QBmsg.Data = []byte{0x01, 00, 00, 00, 00, 00, 00, 00}
	} else {
		// 发送关闭气泵的指令
		QBmsg.Data = []byte{0x00, 00, 00, 00, 00, 00, 00, 00}
	}

	return u.ForwardToCanService("http://localhost:5260", QBmsg)
}

// ControlAirPumpWithCANAsync 使用CAN通信控制气泵（异步版本）
func (u *Utils) ControlAirPumpWithCANAsync(on bool) {
	QBmsg := CanMessage{
		Interface: "can4", // 默认气泵接口
		Id:        0x101,
		Data:      []byte{},
	}

	if on {
		// 发送开启气泵的指令
		QBmsg.Data = []byte{0x01, 00, 00, 00, 00, 00, 00, 00}
	} else {
		// 发送关闭气泵的指令
		QBmsg.Data = []byte{0x00, 00, 00, 00, 00, 00, 00, 00}
	}

	// 异步发送，不等待响应
	u.ForwardToCanServiceAsync("http://localhost:5260", QBmsg)
}

// SwitchFingeringWithLogging 带日志记录的指法切换（保留用于手动发送）
func (u *Utils) SwitchFingeringWithLogging(cfg Config, fingering FingeringEntry, instrument string) error {
	// 创建指法构建器
	fingeringBuilder := NewFingeringBuilder()

	// 根据乐器类型选择不同的配置
	var leftPressProfile, leftReleaseProfile, rightPressProfile, rightReleaseProfile []int

	if instrument == "sn" {
		leftPressProfile = cfg.SnLeftPressProfile
		leftReleaseProfile = cfg.SnLeftReleaseProfile
		rightPressProfile = cfg.SnRightPressProfile
		rightReleaseProfile = cfg.SnRightReleaseProfile
	} else {
		leftPressProfile = cfg.SksLeftPressProfile
		leftReleaseProfile = cfg.SksLeftReleaseProfile
		rightPressProfile = cfg.SksRightPressProfile
		rightReleaseProfile = cfg.SksRightReleaseProfile
	}

	// 生成左右手的CAN数据帧
	leftFrame := fingeringBuilder.BuildFingerFrame(fingering.Left, leftPressProfile, leftReleaseProfile, cfg, instrument)
	rightFrame := fingeringBuilder.BuildFingerFrame(fingering.Right, rightPressProfile, rightReleaseProfile, cfg, instrument)

	// 并发发送左右手指法指令
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// 发送左手指令（即使指法为空也要发送，确保手指释放到正确位置）
	wg.Add(1)
	go func() {
		defer wg.Done()
		leftID := u.ParseCanID(cfg.Hands.Left.ID)
		err := u.SendCanFrame(cfg, cfg.Hands.Left.Interface, leftID, leftFrame)
		if err != nil {
			errChan <- fmt.Errorf("左手指令发送失败: %v", err)
		}
	}()

	// 发送右手指令（即使指法为空也要发送，确保手指释放到正确位置）
	wg.Add(1)
	go func() {
		defer wg.Done()
		rightID := u.ParseCanID(cfg.Hands.Right.ID)
		err := u.SendCanFrame(cfg, cfg.Hands.Right.Interface, rightID, rightFrame)
		if err != nil {
			errChan <- fmt.Errorf("右手指令发送失败: %v", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// 检查发送是否有错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
