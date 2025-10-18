#!/bin/bash
################################################################################
# 远程停止脚本
# 用途：通过HTTP API停止树莓派正在播放的音乐
# 使用：./stop_remote.sh [树莓派IP:端口]
################################################################################

# 默认参数
DEFAULT_HOST="localhost:8088"

# 获取参数
HOST="${1:-$DEFAULT_HOST}"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🛑 准备停止播放...${NC}"
echo -e "目标地址: ${GREEN}$HOST${NC}"

# 发送停止请求
echo -e "\n${YELLOW}📤 发送停止请求...${NC}"
RESPONSE=$(curl -s -X POST "http://$HOST/api/playback/stop")

# 检查响应
if echo "$RESPONSE" | grep -q "error"; then
  echo -e "${RED}❌ 停止失败:${NC}"
  echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
  exit 1
else
  echo -e "${GREEN}✅ 停止成功!${NC}"
  echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
fi

