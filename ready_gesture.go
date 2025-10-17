package main

import (
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// 预备手势模块
////////////////////////////////////////////////////////////////////////////////

// ReadyGestureController 预备手势控制器
type ReadyGestureController struct {
	fingeringBuilder *FingeringBuilder
}

// NewReadyGestureController 创建新的预备手势控制器
func NewReadyGestureController() *ReadyGestureController {
	return &ReadyGestureController{
		fingeringBuilder: NewFingeringBuilder(),
	}
}

// ExecuteReadyGesture 执行预备手势（将所有手指设置为释放状态，支持乐器类型）
func (rgc *ReadyGestureController) ExecuteReadyGesture(cfg Config, instrument string) error {
	// 根据乐器类型选择配置
	var leftReleaseProfile, rightReleaseProfile []int

	if instrument == "sn" {
		leftReleaseProfile = cfg.SnLeftReleaseProfile
		rightReleaseProfile = cfg.SnRightReleaseProfile
	} else {
		leftReleaseProfile = cfg.SksLeftReleaseProfile
		rightReleaseProfile = cfg.SksRightReleaseProfile
	}

	// 构建全释放数据帧
	leftFrame := rgc.fingeringBuilder.BuildReleaseFrame(leftReleaseProfile)
	rightFrame := rgc.fingeringBuilder.BuildReleaseFrame(rightReleaseProfile)

	// 并发发送预备手势
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		utils := NewUtils()
		leftID := utils.ParseCanID(cfg.Hands.Left.ID)
		errChan <- utils.SendCanFrame(cfg, cfg.Hands.Left.Interface, leftID, leftFrame)
	}()

	go func() {
		defer wg.Done()
		utils := NewUtils()
		rightID := utils.ParseCanID(cfg.Hands.Right.ID)
		errChan <- utils.SendCanFrame(cfg, cfg.Hands.Right.Interface, rightID, rightFrame)
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// ExecuteReadyGestureWithDelay 执行预备手势并等待指定时间
func (rgc *ReadyGestureController) ExecuteReadyGestureWithDelay(cfg Config, instrument string, holdMS int) error {
	if err := rgc.ExecuteReadyGesture(cfg, instrument); err != nil {
		return err
	}

	if holdMS > 0 {
		time.Sleep(time.Duration(holdMS) * time.Millisecond)
	}

	return nil
}

// BuildSmoothThumbTransitionFrame 构建唢呐拇指平滑切换的释放指令
func (rgc *ReadyGestureController) BuildSmoothThumbTransitionFrame(leftRelease []int) []byte {
	return rgc.fingeringBuilder.BuildReleaseFrame(leftRelease)
}
