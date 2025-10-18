#!/bin/bash
################################################################################
# 终极停止脚本
# 用途：使用所有可能的方法查找并停止 sksgo 相关进程
# 使用：./kill_all.sh
################################################################################

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🛑 终极停止脚本 - 使用所有方法查找并停止进程...${NC}"
echo ""

STOPPED=0

# 方法1: pkill 按名称模糊匹配
echo -e "${YELLOW}[1] 尝试 pkill -f 方法...${NC}"
if pgrep -f "sksgo|sksarm7" > /dev/null 2>&1; then
  pkill -15 -f "sksgo|sksarm7" 2>/dev/null
  echo -e "  ${GREEN}✓ 已发送停止信号${NC}"
  STOPPED=$((STOPPED+1))
else
  echo -e "  ${YELLOW}- 未找到进程${NC}"
fi

# 方法2: 通过可执行文件名精确匹配
echo -e "${YELLOW}[2] 尝试精确匹配可执行文件名...${NC}"
for name in newsksgo sksgo newsksarm7 sksarm7; do
  if pgrep -x "$name" > /dev/null 2>&1; then
    pkill -15 -x "$name" 2>/dev/null
    echo -e "  ${GREEN}✓ 停止 $name${NC}"
    STOPPED=$((STOPPED+1))
  fi
done

# 方法3: 查找包含特定参数的进程
echo -e "${YELLOW}[3] 尝试查找带 -json/-exec 参数的进程...${NC}"
JSON_PIDS=$(ps aux | grep -E "\-json|\-exec" | grep -v "grep\|kill_all" | awk '{print $2}')
if [ -n "$JSON_PIDS" ]; then
  for pid in $JSON_PIDS; do
    if ps -p $pid > /dev/null 2>&1; then
      kill -15 $pid 2>/dev/null
      echo -e "  ${GREEN}✓ 停止进程 $pid${NC}"
      STOPPED=$((STOPPED+1))
    fi
  done
else
  echo -e "  ${YELLOW}- 未找到进程${NC}"
fi

# 方法4: 停止8088端口的进程
echo -e "${YELLOW}[4] 尝试停止8088端口的进程...${NC}"
PORT_PID=$(lsof -t -i:8088 2>/dev/null)
if [ -n "$PORT_PID" ]; then
  for pid in $PORT_PID; do
    kill -15 $pid 2>/dev/null
    echo -e "  ${GREEN}✓ 停止端口8088进程 $pid${NC}"
    STOPPED=$((STOPPED+1))
  done
else
  echo -e "  ${YELLOW}- 端口未被占用${NC}"
fi

# 方法5: 查找当前目录下运行的进程
echo -e "${YELLOW}[5] 尝试查找从当前目录启动的进程...${NC}"
CURRENT_DIR=$(pwd)
DIR_PIDS=$(ps aux | grep "$CURRENT_DIR" | grep -E "newsksgo|sksgo|sksarm7" | grep -v "grep\|kill_all" | awk '{print $2}')
if [ -n "$DIR_PIDS" ]; then
  for pid in $DIR_PIDS; do
    if ps -p $pid > /dev/null 2>&1; then
      kill -15 $pid 2>/dev/null
      echo -e "  ${GREEN}✓ 停止进程 $pid${NC}"
      STOPPED=$((STOPPED+1))
    fi
  done
else
  echo -e "  ${YELLOW}- 未找到进程${NC}"
fi

# 等待进程退出
if [ $STOPPED -gt 0 ]; then
  echo ""
  echo -e "${YELLOW}⏳ 等待进程退出...${NC}"
  sleep 3

  # 检查是否还有残留进程
  echo -e "${YELLOW}[6] 检查残留进程...${NC}"
  REMAINING=$(pgrep -af "sksgo|sksarm7")
  
  if [ -n "$REMAINING" ]; then
    echo -e "${YELLOW}⚠️  发现残留进程，使用强制停止...${NC}"
    echo "$REMAINING"
    
    # 强制停止
    pkill -9 -f "sksgo|sksarm7" 2>/dev/null
    
    # 停止特定PID
    REMAIN_PIDS=$(echo "$REMAINING" | awk '{print $1}')
    for pid in $REMAIN_PIDS; do
      if ps -p $pid > /dev/null 2>&1; then
        kill -9 $pid 2>/dev/null
        echo -e "  ${RED}✓ 强制停止 $pid${NC}"
      fi
    done
    
    sleep 1
  fi
fi

# 最终验证
echo ""
echo -e "${YELLOW}[7] 最终验证...${NC}"
FINAL=$(pgrep -af "sksgo|sksarm7")
if [ -z "$FINAL" ]; then
  echo -e "${GREEN}✅ 所有进程已成功停止!${NC}"
  exit 0
else
  echo -e "${RED}❌ 仍有进程在运行:${NC}"
  echo "$FINAL"
  echo ""
  echo -e "${YELLOW}💡 手动停止命令:${NC}"
  FINAL_PIDS=$(echo "$FINAL" | awk '{print $1}')
  echo -e "   kill -9 $FINAL_PIDS"
  exit 1
fi

