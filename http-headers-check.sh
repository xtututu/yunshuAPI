#!/bin/bash

echo "=== 服务器HTTP缓存头深度检查 ==="
echo

# 检查服务器返回的缓存头
echo "🌐 1. 检查react-core-DxT2a86c.js的HTTP头:"
echo "Chrome User-Agent:"
curl -v "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
  2>&1 | grep -E "(HTTP|Content-Type|Cache-Control|ETag|Last-Modified|Expires)"

echo
echo "🍎 Safari User-Agent:"
curl -v "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15" \
  2>&1 | grep -E "(HTTP|Content-Type|Cache-Control|ETag|Last-Modified|Expires)"

echo
echo "📱 2. 检查react-components-BXXLEu8q.js的HTTP头:"
curl -v "https://token.yishangcloud.cn/assets/react-components-BXXLEu8q.js" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
  2>&1 | grep -E "(HTTP|Content-Type|Cache-Control|ETag|Last-Modified|Expires)"

echo
echo "🔍 3. 检查index.html中的文件引用:"
curl -s "https://token.yishangcloud.cn/" | grep -E "(react-core-DxT2a86c|react-components-BXXLEu8q)" || echo "未找到文件引用"

echo
echo "=== 检查完成 ==="