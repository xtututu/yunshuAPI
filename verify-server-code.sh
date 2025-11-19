#!/bin/bash

echo "=== 服务器端代码验证脚本 ==="
echo "验证时间: $(date)"
echo ""

# 1. 检查Git状态
echo "1. 检查Git状态："
if ! command -v git &> /dev/null; then
    echo "❌ Git未安装"
    exit 1
fi

git status
echo ""

# 2. 检查当前分支和最新提交
echo "2. 检查当前分支和最新提交："
echo "当前分支: $(git branch --show-current)"
echo "最新提交:"
git log --oneline -3
echo ""

# 3. 检查是否与远程同步
echo "3. 检查与远程仓库的同步状态："
git remote update
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
BASE=$(git merge-base HEAD origin/main)

if [ $LOCAL = $REMOTE ]; then
    echo "✅ 本地代码与远程仓库同步"
elif [ $LOCAL = $BASE ]; then
    echo "⚠️  本地代码落后于远程仓库，需要执行: git pull origin main"
else
    echo "⚠️  本地代码领先于远程仓库"
fi
echo ""

# 4. 检查关键修复是否存在
echo "4. 检查关键修复是否存在："

# 检查router/web-router.go中的assets拦截是否已移除
if [ -f "router/web-router.go" ]; then
    if grep -q "strings.HasPrefix(c.Request.RequestURI, \"/assets\")" router/web-router.go; then
        echo "❌ router/web-router.go中仍存在assets拦截（未修复）"
    else
        echo "✅ router/web-router.go中已移除assets拦截（已修复）"
    fi
else
    echo "❌ router/web-router.go文件不存在"
fi

# 检查vite.config.js中的base配置
if [ -f "web/vite.config.js" ]; then
    if grep -q "base: process.env.NODE_ENV === 'production'" web/vite.config.js; then
        echo "✅ web/vite.config.js中base配置已更新（已修复）"
    else
        echo "❌ web/vite.config.js中base配置未更新（未修复）"
    fi
else
    echo "❌ web/vite.config.js文件不存在"
fi
echo ""

# 5. 检查静态资源文件
echo "5. 检查静态资源文件："
if [ -f "web/dist/assets/react-core-DxT2a86c.js" ]; then
    echo "✅ react-core-DxT2a86c.js 存在"
    echo "   文件大小: $(du -h web/dist/assets/react-core-DxT2a86c.js | cut -f1)"
else
    echo "❌ react-core-DxT2a86c.js 不存在"
fi

if [ -f "web/dist/assets/react-components-BXXLEu8q.js" ]; then
    echo "✅ react-components-BXXLEu8q.js 存在"
    echo "   文件大小: $(du -h web/dist/assets/react-components-BXXLEu8q.js | cut -f1)"
else
    echo "❌ react-components-BXXLEu8q.js 不存在"
fi
echo ""

# 6. 检查Docker相关
echo "6. 检查Docker状态："
if command -v docker &> /dev/null; then
    echo "✅ Docker已安装"
    
    # 检查容器状态
    if command -v docker-compose &> /dev/null; then
        echo "✅ Docker Compose已安装"
        echo "当前运行的容器:"
        docker-compose ps
    else
        echo "⚠️  Docker Compose未安装"
        echo "当前运行的容器:"
        docker ps
    fi
else
    echo "❌ Docker未安装"
fi
echo ""

# 7. 提供操作建议
echo "=== 操作建议 ==="
echo "如果发现问题，请按以下步骤操作："
echo ""
echo "1. 拉取最新代码："
echo "   git pull origin main"
echo ""
echo "2. 重新构建并启动容器："
echo "   docker-compose down"
echo "   docker-compose up --build -d"
echo ""
echo "3. 查看容器日志（如果需要）："
echo "   docker-compose logs -f"
echo ""
echo "4. 清除浏览器缓存后重新访问网站"
echo ""

echo "=== 验证完成 ==="