#!/bin/bash
################################################################################
# 智能预处理脚本（自动提取BPM）
# 用途：自动从文件名提取BPM并生成执行序列
# 文件名格式：xxx-108.json（最后的数字为BPM）
# 使用：./preprocess_auto.sh <音乐文件> [乐器类型] [吐音延迟]
# 示例：./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json sn 20
################################################################################

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 检查参数
if [ $# -lt 1 ]; then
  echo -e "${RED}❌ 错误: 缺少音乐文件参数${NC}"
  echo ""
  echo -e "${YELLOW}用法:${NC}"
  echo -e "  ./preprocess_auto.sh <音乐文件> [乐器类型] [吐音延迟]"
  echo ""
  echo -e "${YELLOW}示例:${NC}"
  echo -e "  ./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json"
  echo -e "  ./preprocess_auto.sh trsmusic/青花瓷-葫芦丝-4min-108.json sn 20"
  echo -e "  ./preprocess_auto.sh trsmusic/茉莉花-92.json sks 25"
  echo ""
  echo -e "${CYAN}说明:${NC}"
  echo -e "  - 乐器类型: sn=唢呐(默认), sks=萨克斯"
  echo -e "  - 吐音延迟: 默认20毫秒"
  echo -e "  - BPM: 自动从文件名提取（例如：xxx-108.json → BPM=108）"
  exit 1
fi

# 获取参数
MUSIC_FILE="$1"
INSTRUMENT="${2:-sn}"
TONGUE="${3:-20}"

# 检查文件是否存在
if [ ! -f "$MUSIC_FILE" ]; then
  echo -e "${RED}❌ 错误: 文件不存在: $MUSIC_FILE${NC}"
  exit 1
fi

# 提取文件名
filename=$(basename "$MUSIC_FILE")

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}            智能预处理（自动BPM识别）${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}输入文件:${NC} $filename"

# 从文件名中提取BPM（最后一个-后面的数字）
# 支持以下格式：
#   - 青花瓷-葫芦丝-4min-108.json → 108
#   - 茉莉花-92.json → 92
#   - 康定情歌-唢呐-100.json → 100
BPM=$(echo "$filename" | grep -oP '\-(\d+)\.json$' | grep -oP '\d+')

if [ -z "$BPM" ]; then
  echo -e "${YELLOW}⚠️  警告: 无法从文件名提取BPM${NC}"
  echo -e "${YELLOW}   文件名格式应为: xxx-数字.json（例如：青花瓷-108.json）${NC}"
  echo ""
  echo -e "${YELLOW}请输入BPM值（默认108）:${NC} "
  read -r USER_BPM
  BPM="${USER_BPM:-108}"
fi

# 显示配置
echo ""
echo -e "${CYAN}配置信息:${NC}"
echo -e "  ${GREEN}✓${NC} BPM: ${GREEN}$BPM${NC}"
echo -e "  ${GREEN}✓${NC} 乐器类型: ${GREEN}$INSTRUMENT${NC} ($([ "$INSTRUMENT" = "sn" ] && echo "唢呐" || echo "萨克斯"))"
echo -e "  ${GREEN}✓${NC} 吐音延迟: ${GREEN}$TONGUE${NC} ms"
echo ""

# 生成输出文件名预览
OUTPUT_NAME=$(basename "$MUSIC_FILE" .json)
OUTPUT_FILE="exec/${OUTPUT_NAME}_${INSTRUMENT}_${BPM}_${TONGUE}.exec.json"
echo -e "${CYAN}将生成文件:${NC} $OUTPUT_FILE"
echo ""

# 确认
read -p "是否继续? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}已取消${NC}"
  exit 0
fi

echo ""
echo -e "${YELLOW}⏳ 正在预处理...${NC}"
echo ""

# 执行预处理
./newsksgo -preprocess \
  -in "$MUSIC_FILE" \
  -instrument "$INSTRUMENT" \
  -bpm "$BPM" \
  -tongue "$TONGUE"

# 检查结果
if [ $? -eq 0 ]; then
  echo ""
  echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
  echo -e "${GREEN}✅ 预处理成功！${NC}"
  echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
  echo ""
  
  # 显示文件信息
  if [ -f "$OUTPUT_FILE" ]; then
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${CYAN}文件信息:${NC}"
    echo -e "  路径: $OUTPUT_FILE"
    echo -e "  大小: $FILE_SIZE"
    echo ""
    
    # 显示播放命令
    echo -e "${YELLOW}播放命令:${NC}"
    echo -e "  ./newsksgo -json $OUTPUT_FILE"
    echo ""
  fi
else
  echo ""
  echo -e "${RED}════════════════════════════════════════════════════════════════${NC}"
  echo -e "${RED}❌ 预处理失败${NC}"
  echo -e "${RED}════════════════════════════════════════════════════════════════${NC}"
  exit 1
fi

