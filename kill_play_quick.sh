#!/bin/bash
################################################################################
# 快速停止播放脚本（无需确认）
# 用途：立即停止所有 newsksgo 进程（优雅停止）
# 使用：./kill_play_quick.sh
################################################################################

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🛑 快速停止所有 sksgo 相关进程...${NC}"

# 查找进程（支持多种变体）
PIDS=$(pgrep -f "sksgo|sksarm7" 2>/dev/null)

# 如果上面没找到，尝试更精确的匹配
if [ -z "$PIDS" ]; then
  PIDS=$(ps aux | grep -E "(newsksgo|sksgo|sksarm7).*(-json|-exec|8088)" | grep -v grep | awk '{print $2}')
fi

if [ -z "$PIDS" ]; then
  echo -e "${YELLOW}ℹ️  没有找到正在运行的进程${NC}"
  exit 0
fi

# 显示进程数量
COUNT=$(echo "$PIDS" | wc -w)
echo -e "找到 ${YELLOW}$COUNT${NC} 个进程"

# 优雅停止（支持多种程序名）
pkill -15 -f "sksgo|sksarm7" 2>/dev/null

# 等待2秒
sleep 2

# 检查是否还有残留
REMAINING=$(pgrep -f "sksgo|sksarm7" 2>/dev/null)
if [ -n "$REMAINING" ]; then
  echo -e "${YELLOW}⚠️  仍有进程未停止，使用强制停止...${NC}"
  pkill -9 -f "sksgo|sksarm7" 2>/dev/null
  sleep 1
fi

# 最终检查
FINAL=$(pgrep -f "sksgo|sksarm7" 2>/dev/null)
if [ -z "$FINAL" ]; then
  echo -e "${GREEN}✅ 所有进程已停止${NC}"
else
  echo -e "${RED}❌ 停止失败，请手动停止: kill -9 $FINAL${NC}"
  exit 1
fi

