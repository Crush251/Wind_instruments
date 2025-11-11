package dump

import (
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial.v1"
)

type PumpController struct {
	port serial.Port
}

func NewPumpController(portName string) (*PumpController, error) {
	mode := &serial.Mode{BaudRate: 9600}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}

	port.ResetInputBuffer()

	return &PumpController{port: port}, nil
}

func (c *PumpController) Close() {
	if c.port != nil {
		c.port.Close()
	}
}

func (c *PumpController) send(cmd string) string {
	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	c.port.Write([]byte(cmd))

	time.Sleep(50 * time.Millisecond)
	buf := make([]byte, 1024)
	n, _ := c.port.Read(buf)
	return string(buf[:n])
}

// 命令方法
func (c *PumpController) Help() string   { return c.send("help") }
func (c *PumpController) Auto() string   { return c.send("auto") }
func (c *PumpController) Manual() string { return c.send("manual") }
func (c *PumpController) On() string     { return c.send("on") }
func (c *PumpController) Off() string    { return c.send("off") }

func (c *PumpController) SetPWM(value int) string {
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	return c.send(fmt.Sprintf("set %d", value))
}

func (c *PumpController) SetSpeed(step int) string {
	if step < 1 {
		step = 1
	} else if step > 50 {
		step = 50
	}
	return c.send(fmt.Sprintf("speed %d", step))
}

func (c *PumpController) Status() map[string]string {
	result := map[string]string{"raw": c.send("status")}
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
