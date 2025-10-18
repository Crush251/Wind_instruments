#!/bin/bash
################################################################################
# Web服务启动脚本
# 用途：启动萨克斯/唢呐演奏Web服务，监听8088端口
# 使用：./start_server.sh [配置文件路径]
################################################################################

# 默认参数
CONFIG_FILE="${1:-config.yaml}"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 启动萨克斯/唢呐演奏Web服务...${NC}"
echo -e "配置文件: ${GREEN}$CONFIG_FILE${NC}"
echo -e "监听端口: ${GREEN}8088${NC}"
echo ""

# 检查是否已有进程在运行
if lsof -Pi :8088 -sTCP:LISTEN -t >/dev/null 2>&1; then
  echo -e "${YELLOW}⚠️  检测到8088端口已被占用${NC}"
  echo -e "是否要停止旧进程? (y/n)"
  read -r answer
  if [ "$answer" = "y" ]; then
    OLD_PID=$(lsof -Pi :8088 -sTCP:LISTEN -t)
    echo -e "${YELLOW}🛑 停止旧进程 PID=$OLD_PID${NC}"
    kill -15 "$OLD_PID" 2>/dev/null
    sleep 2
  else
    echo -e "${YELLOW}❌ 启动取消${NC}"
    exit 1
  fi
fi

# 切换到脚本所在目录
cd "$(dirname "$0")" || exit 1

# 启动服务（后台运行）
echo -e "${GREEN}🎵 启动Web服务...${NC}"
nohup ./newsksgo -config "$CONFIG_FILE" > server.log 2>&1 &
SERVER_PID=$!

# 等待服务启动
sleep 2

# 检查服务是否启动成功
if ps -p $SERVER_PID > /dev/null 2>&1; then
  echo -e "${GREEN}✅ Web服务启动成功!${NC}"
  echo -e "进程ID: ${GREEN}$SERVER_PID${NC}"
  echo -e "访问地址: ${GREEN}http://localhost:8088${NC}"
  echo -e "日志文件: ${GREEN}server.log${NC}"
  echo ""
  echo -e "${YELLOW}提示:${NC}"
  echo -e "  - 查看日志: tail -f server.log"
  echo -e "  - 停止服务: kill $SERVER_PID"
  echo -e "  - 播放音乐: ./play_remote.sh [exec文件名]"
else
  echo -e "${RED}❌ Web服务启动失败，请查看 server.log${NC}"
  exit 1
fi

