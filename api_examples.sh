#!/bin/bash

################################################################################
# 萨克斯/唢呐演奏系统 API 调用示例脚本
# 
# 使用方法：
#   1. 确保Web服务已启动（默认端口：8088）
#   2. 赋予脚本执行权限：chmod +x api_examples.sh
#   3. 运行示例命令
################################################################################

# 服务器地址（可根据实际情况修改）
API_BASE="http://localhost:8088/api"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

################################################################################
# 1. 获取音乐文件列表
################################################################################
echo -e "${BLUE}========== 获取音乐文件列表 ==========${NC}"
get_music_files() {
    echo -e "${GREEN}正在获取音乐文件列表...${NC}"
    curl -s -X GET "${API_BASE}/files" | jq '.'
}

################################################################################
# 2. 开始演奏（萨克斯，使用默认BPM）
################################################################################
echo -e "\n${BLUE}========== 开始演奏示例 ==========${NC}"
start_playback_sax() {
    local filename=$1
    if [ -z "$filename" ]; then
        filename="test.json"
    fi
    
    echo -e "${GREEN}开始演奏萨克斯: ${filename}${NC}"
    curl -s -X POST "${API_BASE}/playback/start" \
        -H "Content-Type: application/json" \
        -d "{
            \"filename\": \"${filename}\",
            \"instrument\": \"sks\",
            \"bpm\": 0,
            \"tonguing_delay\": 30
        }" | jq '.'
}

################################################################################
# 3. 开始演奏（唢呐，自定义BPM和吐音延迟）
################################################################################
start_playback_suona() {
    local filename=$1
    local bpm=$2
    local tonguing_delay=$3
    
    if [ -z "$filename" ]; then
        filename="molihua.json"
    fi
    if [ -z "$bpm" ]; then
        bpm=120
    fi
    if [ -z "$tonguing_delay" ]; then
        tonguing_delay=30
    fi
    
    echo -e "${GREEN}开始演奏唢呐: ${filename} (BPM: ${bpm}, 吐音延迟: ${tonguing_delay}ms)${NC}"
    curl -s -X POST "${API_BASE}/playback/start" \
        -H "Content-Type: application/json" \
        -d "{
            \"filename\": \"${filename}\",
            \"instrument\": \"sn\",
            \"bpm\": ${bpm},
            \"tonguing_delay\": ${tonguing_delay}
        }" | jq '.'
}

################################################################################
# 4. 暂停/恢复演奏
################################################################################
pause_playback() {
    echo -e "${YELLOW}暂停/恢复演奏...${NC}"
    curl -s -X POST "${API_BASE}/playback/pause" \
        -H "Content-Type: application/json" | jq '.'
}

################################################################################
# 5. 停止演奏
################################################################################
stop_playback() {
    echo -e "${RED}停止演奏...${NC}"
    curl -s -X POST "${API_BASE}/playback/stop" \
        -H "Content-Type: application/json" | jq '.'
}

################################################################################
# 6. 获取演奏状态
################################################################################
get_playback_status() {
    echo -e "${BLUE}获取演奏状态...${NC}"
    curl -s -X GET "${API_BASE}/playback/status" | jq '.'
}

################################################################################
# 7. 获取指法映射（萨克斯）
################################################################################
get_fingerings_sax() {
    echo -e "${BLUE}获取萨克斯指法映射...${NC}"
    curl -s -X GET "${API_BASE}/fingerings?instrument=sks" | jq '.'
}

################################################################################
# 8. 获取指法映射（唢呐）
################################################################################
get_fingerings_suona() {
    echo -e "${BLUE}获取唢呐指法映射...${NC}"
    curl -s -X GET "${API_BASE}/fingerings?instrument=sn" | jq '.'
}

################################################################################
# 9. 发送单个指法
################################################################################
send_fingering() {
    local note=$1
    local instrument=$2
    
    if [ -z "$note" ]; then
        note="A4"
    fi
    if [ -z "$instrument" ]; then
        instrument="sks"
    fi
    
    echo -e "${GREEN}发送指法: ${note} (${instrument})${NC}"
    curl -s -X POST "${API_BASE}/fingerings/send" \
        -H "Content-Type: application/json" \
        -d "{
            \"note\": \"${note}\",
            \"instrument\": \"${instrument}\"
        }" | jq '.'
}

################################################################################
# 10. 获取歌曲时间轴
################################################################################
get_timeline() {
    local filename=$1
    if [ -z "$filename" ]; then
        filename="test.json"
    fi
    
    echo -e "${BLUE}获取歌曲时间轴: ${filename}${NC}"
    curl -s -X GET "${API_BASE}/timeline?filename=${filename}" | jq '.'
}

################################################################################
# 11. 更新歌曲时间轴（修改空拍时长示例）
################################################################################
update_timeline() {
    local filename=$1
    
    if [ -z "$filename" ]; then
        filename="test.json"
    fi
    
    echo -e "${YELLOW}更新歌曲时间轴: ${filename}${NC}"
    echo -e "${YELLOW}注意：需要提供完整的timeline数组${NC}"
    
    # 示例：假设要更新的timeline数据
    # 实际使用时需要先获取当前timeline，修改后再提交
    curl -s -X POST "${API_BASE}/timeline/update" \
        -H "Content-Type: application/json" \
        -d "{
            \"filename\": \"${filename}\",
            \"timeline\": [
                [\"A4\", 1],
                [\"B4\", 0.5],
                [\"NO\", 2],
                [\"C5\", 1]
            ]
        }" | jq '.'
}

################################################################################
# 使用示例菜单
################################################################################
show_menu() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}  萨克斯/唢呐演奏系统 API 示例菜单${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo "1.  获取音乐文件列表"
    echo "2.  开始演奏萨克斯（test.json）"
    echo "3.  开始演奏唢呐（molihua.json，BPM 120）"
    echo "4.  暂停/恢复演奏"
    echo "5.  停止演奏"
    echo "6.  获取演奏状态"
    echo "7.  获取萨克斯指法映射"
    echo "8.  获取唢呐指法映射"
    echo "9.  发送单个指法（A4，萨克斯）"
    echo "10. 获取歌曲时间轴（test.json）"
    echo "11. 退出"
    echo -e "${BLUE}========================================${NC}"
    read -p "请选择功能 (1-11): " choice
    
    case $choice in
        1) get_music_files ;;
        2) start_playback_sax "test.json" ;;
        3) start_playback_suona "molihua.json" 120 30 ;;
        4) pause_playback ;;
        5) stop_playback ;;
        6) get_playback_status ;;
        7) get_fingerings_sax ;;
        8) get_fingerings_suona ;;
        9) send_fingering "A4" "sks" ;;
        10) get_timeline "test.json" ;;
        11) echo -e "${GREEN}退出${NC}"; exit 0 ;;
        *) echo -e "${RED}无效选择${NC}" ;;
    esac
}

################################################################################
# 快捷命令示例（作为函数导出，可以直接调用）
################################################################################

# 快速开始：演奏指定文件（萨克斯）
quick_play_sax() {
    echo -e "${GREEN}=== 快速演奏萨克斯 ===${NC}"
    start_playback_sax "$1"
    sleep 2
    echo -e "\n${BLUE}演奏状态：${NC}"
    get_playback_status
}

# 快速开始：演奏指定文件（唢呐）
quick_play_suona() {
    echo -e "${GREEN}=== 快速演奏唢呐 ===${NC}"
    start_playback_suona "$1" "$2" "$3"
    sleep 2
    echo -e "\n${BLUE}演奏状态：${NC}"
    get_playback_status
}

# 快速停止
quick_stop() {
    echo -e "${RED}=== 快速停止演奏 ===${NC}"
    stop_playback
}

################################################################################
# 主程序
################################################################################

# 检查jq是否安装（用于格式化JSON输出）
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}警告: jq未安装，JSON输出可能不美观${NC}"
    echo -e "${YELLOW}安装方法: sudo apt-get install jq${NC}"
fi

# 如果没有参数，显示菜单
if [ $# -eq 0 ]; then
    while true; do
        show_menu
        echo ""
        read -p "按回车继续..." dummy
    done
else
    # 如果有参数，执行对应的命令
    case $1 in
        list) get_music_files ;;
        play-sax) start_playback_sax "$2" ;;
        play-suona) start_playback_suona "$2" "$3" "$4" ;;
        pause) pause_playback ;;
        stop) stop_playback ;;
        status) get_playback_status ;;
        fingerings-sax) get_fingerings_sax ;;
        fingerings-suona) get_fingerings_suona ;;
        send-note) send_fingering "$2" "$3" ;;
        timeline) get_timeline "$2" ;;
        quick-sax) quick_play_sax "$2" ;;
        quick-suona) quick_play_suona "$2" "$3" "$4" ;;
        quick-stop) quick_stop ;;
        *)
            echo -e "${RED}未知命令: $1${NC}"
            echo ""
            echo "用法示例："
            echo "  ./api_examples.sh                           # 交互式菜单"
            echo "  ./api_examples.sh list                      # 列出音乐文件"
            echo "  ./api_examples.sh play-sax test.json        # 演奏萨克斯"
            echo "  ./api_examples.sh play-suona molihua.json 120 30  # 演奏唢呐"
            echo "  ./api_examples.sh stop                      # 停止演奏"
            echo "  ./api_examples.sh status                    # 查看状态"
            echo "  ./api_examples.sh quick-sax test.json       # 快速演奏萨克斯"
            echo "  ./api_examples.sh quick-stop                # 快速停止"
            exit 1
            ;;
    esac
fi


