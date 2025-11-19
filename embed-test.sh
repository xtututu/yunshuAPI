#!/bin/bash

echo "=== 测试Embed文件系统问题 ==="
echo

echo "🔍 1. 检查embed文件系统中的实际文件:"
echo "在容器内测试embed文件系统:"
docker exec yishangyun-api bash -c 'cd /app && find . -name "*.go" -exec grep -l "go:embed" {} \; | head -5'

echo
echo "📁 2. 检查容器内embed文件系统内容:"
docker exec yishangyun-api bash -c 'cd /app && ls -la web/dist/assets/ | head -10'

echo
echo "🌐 3. 测试不同路径的HTTP响应:"
echo "测试 /assets/react-core-DxT2a86c.js:"
docker exec yishangyun-api curl -s -o /dev/null -w "HTTP状态: %{http_code}, 内容大小: %{size_download} bytes\n" http://localhost:3000/assets/react-core-DxT2a86c.js

echo
echo "测试 /web/dist/assets/react-core-DxT2a86c.js:"
docker exec yishangyun-api curl -s -o /dev/null -w "HTTP状态: %{http_code}, 内容大小: %{size_download} bytes\n" http://localhost:3000/web/dist/assets/react-core-DxT2a86c.js

echo
echo "🔍 4. 测试其他静态文件是否正常:"
echo "测试 /logo.png:"
docker exec yishangyun-api curl -s -o /dev/null -w "HTTP状态: %{http_code}, 内容大小: %{size_download} bytes\n" http://localhost:3000/logo.png

echo
echo "测试 /assets/index-BjD6yqvQ.js:"
docker exec yishangyun-api curl -s -o /dev/null -w "HTTP状态: %{http_code}, 内容大小: %{size_download} bytes\n" http://localhost:3000/assets/index-BjD6yqvQ.js

echo
echo "📊 5. 检查gin路由配置:"
docker exec yishangyun-api bash -c 'cd /app && grep -r "assets" router/ || echo "未找到assets路由配置"'

echo
echo "🚀 6. 检查gin静态资源中间件:"
docker exec yishangyun-api bash -c 'cd /app && grep -r "static.Serve" router/ || echo "未找到static.Serve配置"'

echo
echo "=== 测试完成 ==="