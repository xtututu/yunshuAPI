#!/bin/bash

echo "=== 最终诊断：Embed文件系统问题 ==="
echo

echo "🔍 1. 检查文件名大小写和实际文件名:"
echo "本地assets目录中的react相关文件:"
ls -la /Users/xieMac/Desktop/demoProject/dingding/yishangyunApi/web/dist/assets/ | grep -i react

echo
echo "🌐 2. 测试服务器上的实际文件名:"
echo "容器内assets目录中的react相关文件:"
docker exec yishangyun-api ls -la /app/web/dist/assets/ | grep -i react || echo "容器内未找到react文件"

echo
echo "📊 3. 检查文件名哈希是否匹配:"
echo "本地文件:"
ls -la /Users/xieMac/Desktop/demoProject/dingding/yishangyunApi/web/dist/assets/react-*.js

echo
echo "服务器文件:"
docker exec yishangyun-api ls -la /app/web/dist/assets/react-*.js || echo "服务器未找到react文件"

echo
echo "🔍 4. 检查index.html中的引用是否与实际文件名匹配:"
echo "本地index.html中的react引用:"
grep -o 'react-[^"]*\.js' /Users/xieMac/Desktop/demoProject/dingding/yishangyunApi/web/dist/index.html

echo
echo "服务器index.html中的react引用:"
docker exec yishangyun-api grep -o 'react-[^"]*\.js' /app/web/dist/index.html || echo "服务器未找到引用"

echo
echo "🚀 5. 测试embed文件系统直接访问:"
echo "尝试直接从embed文件系统读取文件:"
docker exec yishangyun-api bash -c 'cd /app && go run -c "
package main
import (
    \"embed\"
    \"fmt\"
    \"io/fs\"
)
//go:embed web/dist
var buildFS embed.FS
func main() {
    files, _ := fs.ReadDir(buildFS, \"web/dist/assets\")
    for _, file := range files {
        if len(file.Name()) > 10 && file.Name()[0:6] == \"react\" {
            fmt.Println(\"Found:\", file.Name())
        }
    }
}
"' || echo "无法测试embed文件系统"

echo
echo "=== 诊断完成 ==="