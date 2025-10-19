#!/bin/bash
################################################################################
# 批量预处理脚本
# 用途：批量生成多个音乐文件的执行序列
# 特性：自动从文件名提取BPM（例如：青花瓷-108.json → BPM=108）
# 使用：./batch_preprocess.sh [乐器类型] [吐音延迟]
################################################################################

# 默认参数
DEFAULT_INSTRUMENT="sn"
DEFAULT_TONGUE="20"

# 获取参数
INSTRUMENT="${1:-$DEFAULT_INSTRUMENT}"
TONGUE="${2:-$DEFAULT_TONGUE}"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}            批量预处理音乐文件（智能BPM识别）${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}参数配置:${NC}"
echo -e "  乐器类型: ${GREEN}$INSTRUMENT${NC} (sn=唢呐, sks=萨克斯)"
echo -e "  吐音延迟: ${GREEN}$TONGUE${NC} ms"
echo -e "  ${CYAN}BPM: 自动从文件名提取${NC}"
echo ""

# 查找所有音乐文件
MUSIC_FILES=(trsmusic/*.json)

if [ ${#MUSIC_FILES[@]} -eq 0 ] || [ ! -f "${MUSIC_FILES[0]}" ]; then
  echo -e "${RED}❌ 错误: 在 trsmusic/ 目录中未找到音乐文件${NC}"
  exit 1
fi

echo -e "${YELLOW}找到 ${#MUSIC_FILES[@]} 个音乐文件${NC}"
echo ""

# 确认
echo -e "${YELLOW}将为以下文件生成执行序列:${NC}"
for file in "${MUSIC_FILES[@]}"; do
  basename "$file"
done
echo ""
read -p "是否继续? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}已取消${NC}"
  exit 0
fi

# 统计
SUCCESS=0
FAIL=0
SKIP=0

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}开始批量处理...${NC}"
echo ""

# 批量处理
for file in "${MUSIC_FILES[@]}"; do
  filename=$(basename "$file")
  echo -e "${BLUE}→${NC} 处理: $filename"
  
  # 检查文件是否存在
  if [ ! -f "$file" ]; then
    echo -e "  ${YELLOW}⚠ 跳过（文件不存在）${NC}"
    ((SKIP++))
    continue
  fi
  
  # 从文件名中提取BPM（最后一个-后面的数字）
  # 例如：青花瓷-葫芦丝-4min-108.json → 108
  BPM=$(echo "$filename" | grep -oP '\-(\d+)\.json$' | grep -oP '\d+')
  
  if [ -z "$BPM" ]; then
    echo -e "  ${YELLOW}⚠ 警告: 无法从文件名提取BPM，使用默认值 108${NC}"
    BPM=108
  else
    echo -e "  ${CYAN}📊 提取BPM: $BPM${NC}"
  fi
  
  # 执行预处理
  OUTPUT=$(./newsksgo -preprocess -in "$file" -instrument "$INSTRUMENT" -bpm "$BPM" -tongue "$TONGUE" 2>&1)
  
  if [ $? -eq 0 ]; then
    # 提取生成的文件名
    EXEC_FILE=$(echo "$OUTPUT" | grep -oP "exec/.*?\.exec\.json" | head -1)
    if [ -n "$EXEC_FILE" ]; then
      echo -e "  ${GREEN}✓ 成功: $EXEC_FILE${NC}"
    else
      echo -e "  ${GREEN}✓ 成功${NC}"
    fi
    ((SUCCESS++))
  else
    echo -e "  ${RED}✗ 失败${NC}"
    echo "$OUTPUT" | sed 's/^/    /'
    ((FAIL++))
  fi
  
  echo ""
done

# 汇总
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}处理完成！${NC}"
echo ""
echo -e "统计:"
echo -e "  ${GREEN}成功: $SUCCESS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "  ${YELLOW}跳过: $SKIP${NC}"
echo -e "  总计: ${#MUSIC_FILES[@]}"
echo ""

# 显示生成的文件
if [ $SUCCESS -gt 0 ]; then
  echo -e "${YELLOW}生成的文件:${NC}"
  ls -lht exec/*_${INSTRUMENT}_*_${TONGUE}.exec.json 2>/dev/null | head -20 | while read line; do
    echo "  $line"
  done
  
  # 统计文件总数
  TOTAL_EXEC=$(ls exec/*_${INSTRUMENT}_*_${TONGUE}.exec.json 2>/dev/null | wc -l)
  if [ $TOTAL_EXEC -gt 20 ]; then
    echo -e "  ${CYAN}... 还有 $((TOTAL_EXEC - 20)) 个文件${NC}"
  fi
fi

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"

