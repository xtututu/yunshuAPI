#!/bin/bash

echo "=== 服务器静态资源深度检查 ==="
echo "时间: $(date)"
echo

# 1. 检查容器内文件是否存在
echo "🔍 1. 检查Docker容器内文件..."
docker exec yishangyunApi ls -la /web/dist/assets/ | grep -E "(react-core|react-components)" || echo "❌ 容器内未找到文件"

# 2. 检查容器内文件详细信息
echo
echo "📁 2. 容器内assets目录详情:"
docker exec yishangyunApi ls -la /web/dist/assets/ | head -20

# 3. 检查文件内容大小
echo
echo "📏 3. 检查特定文件大小:"
docker exec yishangyunApi find /web/dist/assets/ -name "*react-core*" -exec ls -lh {} \;
docker exec yishangyunApi find /web/dist/assets/ -name "*react-components*" -exec ls -lh {} \;

# 4. 检查服务器本地文件映射
echo
echo "🗂️ 4. 检查服务器本地文件映射:"
ls -la ./web/dist/assets/ | grep -E "(react-core|react-components)" || echo "❌ 服务器本地未找到文件"

# 5. 对比文件哈希值
echo
echo "🔐 5. 文件完整性检查:"
if [ -f "./web/dist/assets/react-core-DxT2a86c.js" ]; then
    echo "本地react-core-DxT2a86c.js MD5: $(md5sum ./web/dist/assets/react-core-DxT2a86c.js)"
else
    echo "❌ 本地react-core-DxT2a86c.js不存在"
fi

if [ -f "./web/dist/assets/react-components-BXXLEu8q.js" ]; then
    echo "本地react-components-BXXLEu8q.js MD5: $(md5sum ./web/dist/assets/react-components-BXXLEu8q.js)"
else
    echo "❌ 本地react-components-BXXLEu8q.js不存在"
fi

# 6. 测试容器内HTTP访问
echo
echo "🌐 6. 测试容器内HTTP访问:"
docker exec yishangyunApi curl -I "http://localhost:3000/assets/react-core-DxT2a86c.js" 2>/dev/null || echo "❌ 容器内HTTP访问失败"

# 7. 检查Go应用静态资源配置
echo
echo "⚙️ 7. 检查Go应用静态资源配置:"
docker exec yishangyunApi ps aux | grep new-api

echo
echo "=== 检查完成 ==="