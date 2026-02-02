#!/bin/bash

# 构建和部署脚本：在本地构建镜像，然后上传到服务器运行

# 服务器配置
SERVER_IP="your-server-ip"
SERVER_USER="root"
SERVER_PORT="22"

# 项目配置
PROJECT_NAME="new-api"
IMAGE_NAME="new-api:latest"
TAR_FILE="new-api-image.tar"

# 颜色输出
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

echo -e "${GREEN}=== yunshuAPI 构建和部署脚本 ===${NC}"

# 步骤1：本地构建镜像
echo -e "${YELLOW}[1/5] 开始在本地构建镜像...${NC}"
if docker build --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 -t $IMAGE_NAME .; then
    echo -e "${GREEN}✓ 镜像构建成功${NC}"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi

# 步骤2：保存镜像为tar文件
echo -e "${YELLOW}[2/5] 保存镜像为tar文件...${NC}"
if docker save -o $TAR_FILE $IMAGE_NAME; then
    echo -e "${GREEN}✓ 镜像保存成功${NC}"
else
    echo -e "${RED}✗ 镜像保存失败${NC}"
    rm -f $TAR_FILE
    exit 1
fi

# 步骤3：上传tar文件到服务器
echo -e "${YELLOW}[3/5] 上传镜像到服务器...${NC}"
if scp -P $SERVER_PORT $TAR_FILE $SERVER_USER@$SERVER_IP:~/; then
    echo -e "${GREEN}✓ 镜像上传成功${NC}"
else
    echo -e "${RED}✗ 镜像上传失败${NC}"
    rm -f $TAR_FILE
    exit 1
fi

# 步骤4：在服务器上加载镜像并运行
echo -e "${YELLOW}[4/5] 在服务器上部署镜像...${NC}"
DEPLOY_COMMAND=""
DEPLOY_COMMAND+="docker load -i ~/$TAR_FILE && "
DEPLOY_COMMAND+="docker stop $PROJECT_NAME || true && "
DEPLOY_COMMAND+="docker rm $PROJECT_NAME || true && "
DEPLOY_COMMAND+="docker run -d "
DEPLOY_COMMAND+="--name $PROJECT_NAME "
DEPLOY_COMMAND+="--restart always "
DEPLOY_COMMAND+="--network baota_net "
DEPLOY_COMMAND+="--memory 768m --memory-reservation 512m "
DEPLOY_COMMAND+="--cpus 1.5 "
DEPLOY_COMMAND+="-p 3000:3000 "
DEPLOY_COMMAND+="-v ~/yunshuAPI/data:/data "
DEPLOY_COMMAND+="-v ~/yunshuAPI/logs:/app/logs "
DEPLOY_COMMAND+="-v ~/yunshuAPI/web:/app/web "
DEPLOY_COMMAND+="-e SQL_DSN=root:eZ1rW9jR6mU6uP4b@tcp(mysql:3306)/new-api "
DEPLOY_COMMAND+="-e REDIS_CONN_STRING=redis://redis:6379 "
DEPLOY_COMMAND+="-e TZ=Asia/Shanghai "
DEPLOY_COMMAND+="-e ERROR_LOG_ENABLED=true "
DEPLOY_COMMAND+="-e BATCH_UPDATE_ENABLED=false "
DEPLOY_COMMAND+="-e GENERATE_DEFAULT_TOKEN=true "
DEPLOY_COMMAND+="-e TASK_PRICE_PATCH=sora-2,sora-2-hd,sora-2-pro,veo3,veo3-pro,veo3.1,veo3.1-pro "
DEPLOY_COMMAND+="-e GOMEMLIMIT=512MiB "
DEPLOY_COMMAND+="-e GOGC=80 "
DEPLOY_COMMAND+="$IMAGE_NAME --log-dir /app/logs"

if ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "$DEPLOY_COMMAND"; then
    echo -e "${GREEN}✓ 镜像部署成功${NC}"
else
    echo -e "${RED}✗ 镜像部署失败${NC}"
    rm -f $TAR_FILE
    exit 1
fi

# 步骤5：清理临时文件
echo -e "${YELLOW}[5/5] 清理临时文件...${NC}"
rm -f $TAR_FILE
if ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "rm -f ~/$TAR_FILE"; then
    echo -e "${GREEN}✓ 临时文件清理成功${NC}"
else
    echo -e "${YELLOW}⚠ 服务器临时文件清理失败，但部署已完成${NC}"
fi

echo -e "${GREEN}=== 构建和部署完成 ===${NC}"
echo -e "${GREEN}请登录服务器检查容器状态：${NC}"
echo -e "${YELLOW}docker ps -a | grep $PROJECT_NAME${NC}"
echo -e "${GREEN}检查应用状态：${NC}"
echo -e "${YELLOW}curl http://localhost:3000/api/status${NC}"
