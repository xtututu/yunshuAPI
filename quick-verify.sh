#!/bin/bash

echo "=== 快速验证静态资源修复 ==="
echo ""

# 检查最新提交是否包含修复
echo "检查最新提交："
git log --oneline -1 | grep -E "(修复静态资源|assets|404)" && echo "✅ 找到修复相关提交" || echo "⚠️  未找到修复相关提交"
echo ""

# 检查router配置
echo "检查router/web-router.go："
if [ -f "router/web-router.go" ]; then
    if grep -q "/assets" router/web-router.go; then
        echo "❌ 仍存在assets路径拦截"
        grep -n "/assets" router/web-router.go
    else
        echo "✅ assets路径拦截已移除"
    fi
else
    echo "❌ 文件不存在"
fi
echo ""

# 检查静态文件
echo "检查静态资源文件："
[ -f "web/dist/assets/react-core-DxT2a86c.js" ] && echo "✅ react-core-DxT2a86c.js" || echo "❌ react-core-DxT2a86c.js"
[ -f "web/dist/assets/react-components-BXXLEu8q.js" ] && echo "✅ react-components-BXXLEu8q.js" || echo "❌ react-components-BXXLEu8q.js"
echo ""

# 检查Git状态
echo "检查Git同步状态："
git fetch --dry-run 2>&1 | grep -q "up to date" && echo "✅ 与远程同步" || echo "⚠️  需要拉取最新代码"

echo ""
echo "=== 快速检查完成 ==="