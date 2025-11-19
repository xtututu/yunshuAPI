#!/bin/bash

echo "=== 快速服务器验证 ==="
echo

echo "🔍 1. 检查服务器容器内文件是否存在:"
echo "react-core-DxT2a86c.js:"
docker exec new-api test -f /app/web/dist/assets/react-core-DxT2a86c.js && echo "✅ 存在" || echo "❌ 不存在"

echo
echo "react-components-BXXLEu8q.js:"
docker exec new-api test -f /app/web/dist/assets/react-components-BXXLEu8q.js && echo "✅ 存在" || echo "❌ 不存在"

echo
echo "🌐 2. 检查服务器HTTP响应:"
echo "react-core-DxT2a86c.js HTTP状态:"
docker exec new-api curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/assets/react-core-DxT2a86c.js

echo
echo "react-components-BXXLEu8q.js HTTP状态:"
docker exec new-api curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/assets/react-components-BXXLEu8q.js

echo
echo "📊 3. 检查服务器index.html引用:"
docker exec new-api grep -c 'react-core-DxT2a86c.js' /app/web/dist/index.html 2>/dev/null && echo "✅ 引用存在" || echo "❌ 引用不存在"

echo
echo "🚀 4. 如果文件不存在，需要重新构建容器:"
echo "执行以下命令重新部署:"
echo "docker-compose down"
echo "git pull origin main"  
echo "docker-compose up --build -d"

echo
echo "=== 验证完成 ==="