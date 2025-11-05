#!/bin/bash
################################################################################
# 远程演奏脚本
# 用途：通过HTTP API触发树莓派播放预计算的音乐序列
# 使用：./play_remote.sh [exec文件名] [树莓派IP:端口]
################################################################################

# 默认参数
DEFAULT_EXEC_FILE="青花瓷-葫芦丝-4min-108_sn_108_30.exec.json"
DEFAULT_HOST="localhost:8088"

# 获取参数
EXEC_FILE="${1:-$DEFAULT_EXEC_FILE}"
HOST="${2:-$DEFAULT_HOST}"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🎵 准备播放音乐...${NC}"
echo -e "目标地址: ${GREEN}$HOST${NC}"
echo -e "执行文件: ${GREEN}$EXEC_FILE${NC}"

# 构建JSON请求
JSON_DATA=$(cat <<EOF
{
  "exec_file": "$EXEC_FILE"
}
EOF
)

# 发送播放请求
echo -e "\n${YELLOW}📤 发送播放请求...${NC}"
RESPONSE=$(curl -s -X POST \
  "http://$HOST/api/exec/play" \
  -H "Content-Type: application/json" \
  -d "$JSON_DATA")

# 检查响应
if echo "$RESPONSE" | grep -q "error"; then
  echo -e "${RED}❌ 播放失败:${NC}"
  echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
  exit 1
else
  echo -e "${GREEN}✅ 播放成功!${NC}"
  echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
fi

echo -e "\n${GREEN}🎶 音乐正在播放中...${NC}"

