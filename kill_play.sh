#!/bin/bash
################################################################################
# 停止播放进程脚本
# 用途：查找并停止所有正在运行的 newsksgo 播放进程
# 使用：./kill_play.sh [选项]
#       ./kill_play.sh          # 停止所有进程
#       ./kill_play.sh -9       # 强制停止所有进程
#       ./kill_play.sh --list   # 只列出进程，不停止
################################################################################

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认信号
SIGNAL="-15"  # SIGTERM (优雅停止)

# 解析参数
LIST_ONLY=false
if [ "$1" = "--list" ] || [ "$1" = "-l" ]; then
  LIST_ONLY=true
elif [ "$1" = "-9" ]; then
  SIGNAL="-9"  # SIGKILL (强制停止)
fi

echo -e "${YELLOW}🔍 查找 sksgo 相关进程...${NC}"
echo ""

# 查找所有 sksgo 相关进程（包括 newsksgo, sksgo, newsksarm7, sksarm7）
# 使用更宽松的匹配规则，同时排除当前脚本本身
PIDS=$(pgrep -f "sksgo|sksarm7" | grep -v "$$")

# 也尝试通过二进制文件名查找
if [ -z "$PIDS" ]; then
  PIDS=$(pgrep -x "newsksgo|sksgo|newsksarm7|sksarm7" 2>/dev/null)
fi

# 尝试更宽松的匹配：查找包含 -json 或 -exec 参数的进程
if [ -z "$PIDS" ]; then
  PIDS=$(ps aux | grep -E "(newsksgo|sksgo|sksarm7).*(-json|-exec)" | grep -v grep | awk '{print $2}')
fi

if [ -z "$PIDS" ]; then
  echo -e "${YELLOW}ℹ️  没有找到正在运行的 sksgo 相关进程${NC}"
  echo -e "${BLUE}💡 提示: 手动查找进程${NC}"
  echo -e "   ps aux | grep sksgo"
  exit 0
fi

# 显示进程信息
echo -e "${BLUE}找到以下进程:${NC}"
echo ""
printf "%-10s %-60s %s\n" "PID" "命令行" "运行时间"
echo "--------------------------------------------------------------------------------"

for pid in $PIDS; do
  # 获取进程信息
  if ps -p $pid > /dev/null 2>&1; then
    CMD=$(ps -p $pid -o args= | cut -c 1-58)
    ETIME=$(ps -p $pid -o etime= | xargs)
    
    # 判断进程类型
    if echo "$CMD" | grep -q "\-json\|\-exec"; then
      TYPE="[命令行播放]"
      COLOR=$GREEN
    elif echo "$CMD" | grep -q "8088\|web"; then
      TYPE="[Web服务]"
      COLOR=$BLUE
    else
      TYPE="[未知]"
      COLOR=$YELLOW
    fi
    
    printf "${COLOR}%-10s${NC} %-60s %s\n" "$pid" "$CMD..." "$ETIME"
  fi
done

echo ""

# 如果只是列出，不执行停止
if [ "$LIST_ONLY" = true ]; then
  echo -e "${YELLOW}💡 提示: 要停止这些进程，运行以下命令:${NC}"
  echo -e "   ./kill_play.sh       # 优雅停止"
  echo -e "   ./kill_play.sh -9    # 强制停止"
  exit 0
fi

# 询问确认
echo -e "${YELLOW}⚠️  准备停止上述进程...${NC}"
if [ "$SIGNAL" = "-9" ]; then
  echo -e "${RED}使用强制停止（SIGKILL）${NC}"
else
  echo -e "使用优雅停止（SIGTERM）"
fi

read -p "是否继续? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}❌ 已取消${NC}"
  exit 0
fi

# 停止所有进程
echo ""
echo -e "${YELLOW}🛑 正在停止进程...${NC}"

for pid in $PIDS; do
  if ps -p $pid > /dev/null 2>&1; then
    echo -e "${BLUE}→${NC} 停止进程 $pid"
    kill $SIGNAL $pid 2>/dev/null
    
    if [ $? -eq 0 ]; then
      echo -e "  ${GREEN}✓ 已发送停止信号${NC}"
    else
      echo -e "  ${RED}✗ 停止失败${NC}"
    fi
  fi
done

# 等待进程退出
echo ""
echo -e "${YELLOW}⏳ 等待进程退出...${NC}"
sleep 2

# 检查是否还有残留进程
REMAINING=$(pgrep -f newsksgo)
if [ -z "$REMAINING" ]; then
  echo -e "${GREEN}✅ 所有进程已停止${NC}"
else
  echo -e "${RED}⚠️  仍有进程在运行:${NC}"
  ps -p $REMAINING -o pid,etime,cmd
  echo ""
  echo -e "${YELLOW}💡 提示: 使用强制停止命令:${NC}"
  echo -e "   ./kill_play.sh -9"
  echo -e "   或者: kill -9 $REMAINING"
fi

# 检查8088端口
if lsof -Pi :8088 -sTCP:LISTEN -t >/dev/null 2>&1; then
  PORT_PID=$(lsof -t -i:8088)
  echo -e "${YELLOW}⚠️  警告: 端口8088仍被占用（PID: $PORT_PID）${NC}"
fi

