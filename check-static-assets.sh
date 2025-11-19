#!/bin/bash

echo "=== 检查静态资源修复状态 ==="

# 检查git状态
echo "1. 检查Git状态："
git status
echo ""

# 检查最新提交
echo "2. 检查最新提交："
git log --oneline -3
echo ""

# 检查关键文件是否存在
echo "3. 检查静态资源文件："
if [ -f "web/dist/assets/react-core-DxT2a86c.js" ]; then
    echo "✓ react-core-DxT2a86c.js 存在"
else
    echo "✗ react-core-DxT2a86c.js 不存在"
fi

if [ -f "web/dist/assets/react-components-BXXLEu8q.js" ]; then
    echo "✓ react-components-BXXLEu8q.js 存在"
else
    echo "✗ react-components-BXXLEu8q.js 不存在"
fi
echo ""

# 检查路由配置
echo "4. 检查路由配置："
if grep -q "strings.HasPrefix(c.Request.RequestURI, \"/assets\")" router/web-router.go; then
    echo "✗ 路由配置中仍有assets拦截（需要修复）"
else
    echo "✓ 路由配置中已移除assets拦截"
fi
echo ""

# 检查HTML中的资源引用
echo "5. 检查HTML中的资源引用："
if grep -q "react-core-DxT2a86c.js" web/dist/index.html; then
    echo "✓ HTML中引用了react-core-DxT2a86c.js"
else
    echo "✗ HTML中未引用react-core-DxT2a86c.js"
fi

if grep -q "react-components-BXXLEu8q.js" web/dist/index.html; then
    echo "✓ HTML中引用了react-components-BXXLEu8q.js"
else
    echo "✗ HTML中未引用react-components-BXXLEu8q.js"
fi
echo ""

echo "=== 检查完成 ==="
echo "如果所有检查都通过，请确保："
echo "1. 服务器已拉取最新代码：git pull origin main"
echo "2. Docker容器已重新构建：docker-compose up --build -d"
echo "3. 清除浏览器缓存后重新访问"