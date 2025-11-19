#!/bin/bash

echo "=== 浏览器缓存和HTTP头检查 ==="
echo

# 测试不同HTTP头的响应
echo "🌐 1. 测试静态资源HTTP头:"
curl -I "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
  2>/dev/null | head -10

echo
echo "🍎 2. Safari User-Agent测试:"
curl -I "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15" \
  2>/dev/null | head -10

echo
echo "📱 3. 无缓存测试:"
curl -I "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  -H "Cache-Control: no-cache" \
  -H "Pragma: no-cache" \
  2>/dev/null | head -10

echo
echo "🔍 4. 检查Content-Type和缓存头:"
curl -v "https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js" \
  2>&1 | grep -E "(Content-Type|Cache-Control|ETag|Last-Modified)"

echo
echo "=== 检查完成 ==="