package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////
// 音乐文件扫描器模块
////////////////////////////////////////////////////////////////////////////////

// MusicFileScanner 音乐文件扫描器
type MusicFileScanner struct {
	fileReader *FileReader
}

// NewMusicFileScanner 创建新的音乐文件扫描器
func NewMusicFileScanner() *MusicFileScanner {
	return &MusicFileScanner{
		fileReader: NewFileReader(),
	}
}

// ScanMusicFiles 扫描音乐文件夹
func (mfs *MusicFileScanner) ScanMusicFiles(dir string, search string) ([]MusicFileInfo, error) {
	var files []MusicFileInfo

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理JSON文件
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}

		// 搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(d.Name()), strings.ToLower(search)) {
			return nil
		}

		fileInfo := mfs.ExtractMusicFileInfo(path)
		if fileInfo != nil {
			files = append(files, *fileInfo)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 按文件名排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].Filename < files[j].Filename
	})

	return files, nil
}

// ExtractMusicFileInfo 提取音乐文件信息
func (mfs *MusicFileScanner) ExtractMusicFileInfo(fpath string) *MusicFileInfo {
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil
	}

	var timeline TimelineFile
	if err := json.Unmarshal(data, &timeline); err != nil {
		return nil
	}

	// 获取文件信息
	stat, err := os.Stat(fpath)
	if err != nil {
		return nil
	}

	// 提取元数据
	title := filepath.Base(fpath)
	var bpm float64 = 60 // 默认BPM

	if timeline.Meta != nil {
		if t, ok := timeline.Meta["title"].(string); ok {
			title = t
		}
		utils := NewUtils()
		if b, ok := utils.ConvertToFloat(timeline.Meta["bpm"]); ok && b > 0 {
			bpm = b
		}
	}

	return &MusicFileInfo{
		Filename:   filepath.Base(fpath),
		Title:      title,
		BPM:        bpm,
		Duration:   len(timeline.Timeline),
		FilePath:   fpath,
		FileSize:   stat.Size(),
		ModifiedAt: stat.ModTime().Format("2006-01-02 15:04:05"),
	}
}

// GetMusicFileList 获取音乐文件列表（用于Web API）
func (mfs *MusicFileScanner) GetMusicFileList(dir string, search string) ([]MusicFileInfo, error) {
	return mfs.ScanMusicFiles(dir, search)
}

// ValidateMusicFile 验证音乐文件格式
func (mfs *MusicFileScanner) ValidateMusicFile(fpath string) error {
	// 检查文件是否存在
	if err := mfs.fileReader.CheckFileExists(fpath); err != nil {
		return err
	}

	// 尝试解析文件内容
	data, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("无法读取文件: %v", err)
	}

	var timeline TimelineFile
	if err := json.Unmarshal(data, &timeline); err != nil {
		return fmt.Errorf("文件格式错误: %v", err)
	}

	if len(timeline.Timeline) == 0 {
		return fmt.Errorf("时间轴为空")
	}

	return nil
}
