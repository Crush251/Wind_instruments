#!/bin/bash

################################################################################
# 萨克斯/唢呐演奏控制脚本
# 
# 使用方法：
#   ./play.sh play <音乐文件> [乐器类型] [BPM] [吐音延迟]
#   ./play.sh stop
# 
# 示例：
#   ./play.sh play test.json                    # 萨克斯，默认BPM
#   ./play.sh play test.json sks 120 30         # 萨克斯，120 BPM，30ms吐音
#   ./play.sh play molihua.json sn 100 30       # 唢呐，100 BPM，30ms吐音
#   ./play.sh stop                              # 停止演奏
################################################################################

API_BASE="http://localhost:8088/api"

# 检查参数
if [ $# -lt 1 ]; then
    echo "用法："
    echo "  ./play.sh play <音乐文件> [乐器类型] [BPM] [吐音延迟]"
    echo "  ./play.sh stop"
    echo ""
    echo "示例："
    echo "  ./play.sh play test.json                    # 萨克斯，默认BPM"
    echo "  ./play.sh play test.json sks 120 30         # 萨克斯，120 BPM，30ms吐音"
    echo "  ./play.sh play molihua.json sn 100 30       # 唢呐，100 BPM，30ms吐音"
    echo "  ./play.sh stop                              # 停止演奏"
    exit 1
fi

command=$1

case $command in
    play)
        if [ -z "$2" ]; then
            echo "错误：需要指定音乐文件"
            echo "用法：./play.sh play <音乐文件> [乐器类型] [BPM] [吐音延迟]"
            exit 1
        fi
        
        filename=$2
        instrument=${3:-sks}      # 默认萨克斯
        bpm=${4:-0}               # 默认使用文件BPM
        tonguing_delay=${5:-30}   # 默认30ms
        
        echo "🎵 开始演奏: $filename"
        echo "   乐器: $instrument (sks=萨克斯, sn=唢呐)"
        echo "   BPM: $bpm (0=使用文件默认值)"
        echo "   吐音延迟: ${tonguing_delay}ms"
        
        curl -s -X POST "${API_BASE}/playback/start" \
            -H "Content-Type: application/json" \
            -d "{
                \"filename\": \"${filename}\",
                \"instrument\": \"${instrument}\",
                \"bpm\": ${bpm},
                \"tonguing_delay\": ${tonguing_delay}
            }" | python3 -m json.tool 2>/dev/null || echo ""
        ;;
        
    stop)
        echo "⏹️  停止演奏"
        curl -s -X POST "${API_BASE}/playback/stop" \
            -H "Content-Type: application/json" | python3 -m json.tool 2>/dev/null || echo ""
        ;;
        
    *)
        echo "错误：未知命令 '$command'"
        echo "支持的命令：play, stop"
        exit 1
        ;;
esac


