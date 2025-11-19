#!/bin/bash

# 批量替换import路径的脚本
echo "开始批量替换import路径..."

# 查找所有.go文件并替换import路径
find . -name "*.go" -type f -exec sed -i '' 's|github\.com/QuantumNous/new-api|yishangyunApi|g' {} \;

echo "替换完成！"
echo "检查替换结果..."
echo ""

# 随机检查几个文件的替换结果
echo "=== 检查 main.go ==="
head -n 25 main.go | grep import -A 20

echo ""
echo "=== 检查 router/web-router.go ==="
head -n 15 router/web-router.go | grep import -A 10

echo ""
echo "=== 检查 service/channel.go ==="
head -n 20 service/channel.go | grep import -A 15

echo ""
echo "批量替换完成！请检查上述结果确认替换正确。"