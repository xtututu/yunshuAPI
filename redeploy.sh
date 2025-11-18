#!/bin/bash

echo "开始重新部署应用..."

# 进入项目目录
cd /root/yishangyunApi

# 拉取最新代码
echo "拉取最新代码..."
git pull origin main

# 停止并删除现有容器
echo "停止现有容器..."
docker-compose down

# 重新构建并启动容器
echo "重新构建并启动容器..."
docker-compose up --build -d

# 等待容器启动
echo "等待容器启动..."
sleep 10

# 检查容器状态
echo "检查容器状态..."
docker-compose ps

# 检查日志中的文件系统选择信息
echo "检查静态文件服务配置..."
docker-compose logs yishangyun-api | grep -E "(file system|static files|Docker)"

# 验证静态文件访问
echo "验证静态文件访问..."
curl -I http://localhost:8080/assets/react-core-DxT2a86c.js
curl -I http://localhost:8080/assets/index-D8Z2T2wK.js

echo "部署完成！"
echo "应用地址: http://localhost:8080"