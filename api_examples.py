#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
萨克斯/唢呐演奏系统 API 调用示例（Python版本）

使用方法：
    python3 api_examples.py list                    # 列出音乐文件
    python3 api_examples.py play test.json          # 演奏萨克斯
    python3 api_examples.py play-suona molihua.json 120  # 演奏唢呐（指定BPM）
    python3 api_examples.py stop                    # 停止演奏
    python3 api_examples.py status                  # 查看状态
"""

import sys
import json
import requests
from typing import Optional

# 服务器配置
API_BASE = "http://localhost:8088/api"

class MusicController:
    """音乐演奏控制器"""
    
    def __init__(self, base_url: str = API_BASE):
        self.base_url = base_url
    
    def get_music_files(self, search: str = "") -> dict:
        """获取音乐文件列表"""
        url = f"{self.base_url}/files"
        params = {"search": search} if search else {}
        response = requests.get(url, params=params)
        response.raise_for_status()
        return response.json()
    
    def start_playback(
        self, 
        filename: str, 
        instrument: str = "sks", 
        bpm: float = 0, 
        tonguing_delay: int = 30
    ) -> dict:
        """开始演奏
        
        Args:
            filename: 音乐文件名（如 test.json）
            instrument: 乐器类型（sks=萨克斯, sn=唢呐）
            bpm: 节拍速度（0表示使用文件默认值）
            tonguing_delay: 吐音延迟（毫秒）
        """
        url = f"{self.base_url}/playback/start"
        data = {
            "filename": filename,
            "instrument": instrument,
            "bpm": bpm,
            "tonguing_delay": tonguing_delay
        }
        response = requests.post(url, json=data)
        response.raise_for_status()
        return response.json()
    
    def pause_playback(self) -> dict:
        """暂停/恢复演奏"""
        url = f"{self.base_url}/playback/pause"
        response = requests.post(url)
        response.raise_for_status()
        return response.json()
    
    def stop_playback(self) -> dict:
        """停止演奏"""
        url = f"{self.base_url}/playback/stop"
        response = requests.post(url)
        response.raise_for_status()
        return response.json()
    
    def get_playback_status(self) -> dict:
        """获取演奏状态"""
        url = f"{self.base_url}/playback/status"
        response = requests.get(url)
        response.raise_for_status()
        return response.json()
    
    def get_fingerings(self, instrument: str = "sks") -> dict:
        """获取指法映射
        
        Args:
            instrument: 乐器类型（sks=萨克斯, sn=唢呐）
        """
        url = f"{self.base_url}/fingerings"
        params = {"instrument": instrument}
        response = requests.get(url, params=params)
        response.raise_for_status()
        return response.json()
    
    def send_fingering(self, note: str, instrument: str = "sks") -> dict:
        """发送单个指法
        
        Args:
            note: 音符名称（如 A4）
            instrument: 乐器类型（sks=萨克斯, sn=唢呐）
        """
        url = f"{self.base_url}/fingerings/send"
        data = {
            "note": note,
            "instrument": instrument
        }
        response = requests.post(url, json=data)
        response.raise_for_status()
        return response.json()
    
    def get_timeline(self, filename: str) -> dict:
        """获取歌曲时间轴
        
        Args:
            filename: 音乐文件名（如 test.json）
        """
        url = f"{self.base_url}/timeline"
        params = {"filename": filename}
        response = requests.get(url, params=params)
        response.raise_for_status()
        return response.json()
    
    def update_timeline(self, filename: str, timeline: list) -> dict:
        """更新歌曲时间轴
        
        Args:
            filename: 音乐文件名
            timeline: 完整的时间轴数据（[[note, duration], ...]）
        """
        url = f"{self.base_url}/timeline/update"
        data = {
            "filename": filename,
            "timeline": timeline
        }
        response = requests.post(url, json=data)
        response.raise_for_status()
        return response.json()


def print_json(data: dict):
    """美化打印JSON数据"""
    print(json.dumps(data, ensure_ascii=False, indent=2))


def main():
    """主函数"""
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    
    controller = MusicController()
    command = sys.argv[1]
    
    try:
        if command == "list":
            # 列出音乐文件
            search = sys.argv[2] if len(sys.argv) > 2 else ""
            result = controller.get_music_files(search)
            print(f"共找到 {result['total']} 个音乐文件：")
            for file in result['files']:
                print(f"  📁 {file['filename']} - {file['title']} ({file['bpm']} BPM)")
        
        elif command == "play":
            # 演奏萨克斯
            if len(sys.argv) < 3:
                print("错误：需要指定音乐文件名")
                print("用法：python3 api_examples.py play <filename> [bpm] [tonguing_delay]")
                sys.exit(1)
            
            filename = sys.argv[2]
            bpm = float(sys.argv[3]) if len(sys.argv) > 3 else 0
            tonguing_delay = int(sys.argv[4]) if len(sys.argv) > 4 else 30
            
            result = controller.start_playback(filename, "sks", bpm, tonguing_delay)
            print(f"✅ {result['message']}")
            print("📊 演奏状态：")
            print_json(controller.get_playback_status())
        
        elif command == "play-suona":
            # 演奏唢呐
            if len(sys.argv) < 3:
                print("错误：需要指定音乐文件名")
                print("用法：python3 api_examples.py play-suona <filename> [bpm] [tonguing_delay]")
                sys.exit(1)
            
            filename = sys.argv[2]
            bpm = float(sys.argv[3]) if len(sys.argv) > 3 else 0
            tonguing_delay = int(sys.argv[4]) if len(sys.argv) > 4 else 30
            
            result = controller.start_playback(filename, "sn", bpm, tonguing_delay)
            print(f"✅ {result['message']}")
            print("📊 演奏状态：")
            print_json(controller.get_playback_status())
        
        elif command == "pause":
            # 暂停/恢复
            result = controller.pause_playback()
            print(f"✅ {result['message']}")
        
        elif command == "stop":
            # 停止演奏
            result = controller.stop_playback()
            print(f"✅ {result['message']}")
        
        elif command == "status":
            # 获取状态
            status = controller.get_playback_status()
            print("📊 演奏状态：")
            print_json(status)
        
        elif command == "fingerings":
            # 获取指法
            instrument = sys.argv[2] if len(sys.argv) > 2 else "sks"
            result = controller.get_fingerings(instrument)
            print(f"🎹 {instrument.upper()} 指法映射：")
            for fingering in result['fingerings'][:10]:  # 只显示前10个
                print(f"  {fingering['note']}: L={fingering['left']}, R={fingering['right']}")
            print(f"  ... 共 {len(result['fingerings'])} 个指法")
        
        elif command == "send-note":
            # 发送单个指法
            if len(sys.argv) < 3:
                print("错误：需要指定音符名称")
                print("用法：python3 api_examples.py send-note <note> [instrument]")
                sys.exit(1)
            
            note = sys.argv[2]
            instrument = sys.argv[3] if len(sys.argv) > 3 else "sks"
            result = controller.send_fingering(note, instrument)
            print(f"✅ {result['message']}")
        
        elif command == "timeline":
            # 获取时间轴
            if len(sys.argv) < 3:
                print("错误：需要指定音乐文件名")
                print("用法：python3 api_examples.py timeline <filename>")
                sys.exit(1)
            
            filename = sys.argv[2]
            result = controller.get_timeline(filename)
            print(f"📊 {filename} 时间轴：")
            print(f"  BPM: {result['bpm']}")
            print(f"  音符数量: {len(result['timeline'])}")
            print("  前10个音符：")
            for i, item in enumerate(result['timeline'][:10]):
                print(f"    {i+1}. {item[0]} - {item[1]}拍")
        
        else:
            print(f"错误：未知命令 '{command}'")
            print(__doc__)
            sys.exit(1)
    
    except requests.exceptions.RequestException as e:
        print(f"❌ API请求失败: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()


