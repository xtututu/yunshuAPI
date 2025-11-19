#!/bin/bash

echo "=== 服务器文件同步状态检查 ==="
echo

# 检查服务器容器内的文件是否存在
echo "🔍 1. 检查服务器容器内是否存在目标文件:"
echo "react-core-DxT2a86c.js:"
docker exec yishangyun-api ls -la /app/web/dist/assets/react-core-DxT2a86c.js 2>/dev/null && echo "✅ 文件存在" || echo "❌ 文件不存在"

echo
echo "react-components-BXXLEu8q.js:"
docker exec yishangyun-api ls -la /app/web/dist/assets/react-components-BXXLEu8q.js 2>/dev/null && echo "✅ 文件存在" || echo "❌ 文件不存在"

echo
echo "📊 2. 检查服务器容器内index.html的引用:"
docker exec yishangyun-api grep -E "(react-core-DxT2a86c|react-components-BXXLEu8q)" /app/web/dist/index.html || echo "❌ index.html中未找到引用"

echo
echo "🔄 3. 检查容器构建时间和文件时间戳:"
echo "容器创建时间:"
docker inspect yishangyun-api --format='{{.Created}}'

echo
echo "容器内文件修改时间:"
docker exec yishangyun-api stat -c "%y" /app/web/dist/assets/react-core-DxT2a86c.js 2>/dev/null || echo "❌ 文件不存在"
docker exec yishangyun-api stat -c "%y" /app/web/dist/assets/react-components-BXXLEu8q.js 2>/dev/null || echo "❌ 文件不存在"

echo
echo "📁 4. 检查容器内assets目录的js文件列表:"
docker exec yishangyun-api ls -la /app/web/dist/assets/*.js | grep -E "(react-core|react-components)" || echo "❌ 未找到目标js文件"

echo
echo "🌐 5. 测试从容器内直接访问:"
docker exec yishangyun-api curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/assets/react-core-DxT2a86c.js 2>/dev/null || echo "❌ 容器内访问失败"
docker exec yishangyun-api curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/assets/react-components-BXXLEu8q.js 2>/dev/null || echo "❌ 容器内访问失败"

echo
echo "=== 诊断完成 ==="