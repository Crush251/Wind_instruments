#!/bin/bash
################################################################################
# Web服务停止脚本
# 用途：停止运行在8088端口的萨克斯/唢呐演奏Web服务
# 使用：./stop_server.sh
################################################################################

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🛑 停止Web服务...${NC}"

# 方法1：通过端口号查找进程
PORT_PID=$(lsof -t -i:8088 2>/dev/null)

if [ -n "$PORT_PID" ]; then
  echo -e "${YELLOW}📍 找到监听8088端口的进程: PID=$PORT_PID${NC}"
  kill -15 "$PORT_PID" 2>/dev/null
  sleep 2
  
  # 检查进程是否还在运行
  if ps -p "$PORT_PID" > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  进程未响应，使用强制终止...${NC}"
    kill -9 "$PORT_PID" 2>/dev/null
    sleep 1
  fi
  
  # 再次检查
  if lsof -Pi :8088 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${RED}❌ 停止失败，端口仍被占用${NC}"
    exit 1
  else
    echo -e "${GREEN}✅ Web服务已停止（PID: $PORT_PID）${NC}"
  fi
else
  # 方法2：通过进程名查找
  NEWSKSGO_PIDS=$(pgrep -f newsksgo)
  
  if [ -n "$NEWSKSGO_PIDS" ]; then
    echo -e "${YELLOW}📍 找到newsksgo进程: $NEWSKSGO_PIDS${NC}"
    pkill -15 -f newsksgo 2>/dev/null
    sleep 2
    
    # 检查是否还有残留进程
    if pgrep -f newsksgo > /dev/null 2>&1; then
      echo -e "${YELLOW}⚠️  进程未响应，使用强制终止...${NC}"
      pkill -9 -f newsksgo 2>/dev/null
    fi
    
    echo -e "${GREEN}✅ Web服务已停止${NC}"
  else
    echo -e "${YELLOW}ℹ️  没有找到正在运行的Web服务${NC}"
  fi
fi

# 最终检查
if lsof -Pi :8088 -sTCP:LISTEN -t >/dev/null 2>&1; then
  NEW_PID=$(lsof -t -i:8088)
  echo -e "${RED}⚠️  警告: 端口8088仍被占用（PID: $NEW_PID）${NC}"
  echo -e "${YELLOW}手动停止命令: kill -9 $NEW_PID${NC}"
  exit 1
else
  echo -e "${GREEN}✅ 端口8088已释放${NC}"
fi

