#!/bin/bash

echo "=== 服务器文件内容深度检查 ==="
echo

# 检查服务器上实际部署的文件
echo "🔍 1. 检查服务器容器内的文件列表:"
docker exec yishangyun-api ls -la /app/web/dist/assets/ | grep -E "(react-core|react-components)" || echo "容器内未找到目标文件"

echo
echo "📁 2. 检查容器内完整的assets目录:"
docker exec yishangyun-api ls -la /app/web/dist/assets/ | head -20

echo
echo "🌐 3. 直接从容器内访问文件测试:"
docker exec yishangyun-api curl -I http://localhost:3000/assets/react-core-DxT2a86c.js 2>/dev/null || echo "容器内访问失败"

echo
echo "📊 4. 检查文件大小和哈希值:"
docker exec yishangyun-api find /app/web/dist/assets/ -name "*.js" -exec ls -la {} \; | head -10

echo
echo "🔍 5. 检查index.html中的实际引用:"
docker exec yishangyun-api cat /app/web/dist/index.html | grep -E "(react-core|react-components)" || echo "index.html中未找到引用"

echo
echo "📋 6. 对比本地和服务器文件名:"
echo "本地文件:"
ls -la web/dist/assets/ | grep -E "(react-core|react-components)" || echo "本地未找到目标文件"

echo
echo "服务器文件:"
docker exec yishangyun-api ls -la /app/web/dist/assets/ | grep -E "(react-core|react-components)" || echo "服务器未找到目标文件"

echo
echo "=== 检查完成 ==="