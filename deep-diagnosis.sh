#!/bin/bash

echo "=== 深度诊断跨设备404问题 ==="
echo

echo "🔍 1. 检查服务器HTTP响应状态和头部:"
echo "检查react-core-DxT2a86c.js:"
curl -I "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" 2>/dev/null | head -10

echo
echo "检查react-components-BXXLEu8q.js:"
curl -I "https://token.yishangcloud.cn/assets/react-components-BXXLEu8q.js" 2>/dev/null | head -10

echo
echo "🌐 2. 检查服务器返回的实际内容:"
echo "react-core-DxT2a86c.js前100字符:"
curl -s "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" 2>/dev/null | head -c 100
echo ""

echo
echo "react-components-BXXLEu8q.js前100字符:"
curl -s "https://token.yishangcloud.cn/assets/react-components-BXXLEu8q.js" 2>/dev/null | head -c 100
echo ""

echo
echo "🔧 3. 检查服务器容器内gin服务状态:"
docker exec yishangyun-api ps aux | grep gin || echo "未找到gin进程"

echo
echo "📡 4. 检查容器内端口监听状态:"
docker exec yishangyun-api netstat -tlnp | grep :3000 || echo "端口3000未监听"

echo
echo "🌍 5. 检查容器内HTTP服务响应:"
docker exec yishangyun-api curl -I http://localhost:3000/assets/react-core-DxT2a86c.js 2>/dev/null | head -5

echo
echo "📁 6. 检查文件权限和大小:"
docker exec yishangyun-api ls -la /app/web/dist/assets/react-core-DxT2a86c.js 2>/dev/null || echo "文件不存在"
docker exec yishangyun-api ls -la /app/web/dist/assets/react-components-BXXLEu8q.js 2>/dev/null || echo "文件不存在"

echo
echo "🔍 7. 检查Docker网络配置:"
docker network ls | grep yishangyun || echo "未找到相关网络"

echo
echo "📊 8. 检查容器环境变量:"
docker exec yishangyun-api env | grep -E "(PORT|HOST|FRONTEND)" || echo "未找到相关环境变量"

echo
echo "🚀 9. 检查容器启动日志中的错误:"
docker logs yishangyun-api 2>&1 | tail -20 | grep -i error || echo "未发现错误日志"

echo
echo "=== 诊断完成 ==="