#!/bin/bash

echo "=== 强制清除浏览器缓存的解决方案 ==="
echo

echo "🔧 问题分析："
echo "- 本地访问正常"
echo "- Safari正常" 
echo "- Chrome 404"
echo "- 服务器文件存在且正常"
echo

echo "🎯 这99%是浏览器缓存问题！"
echo

echo "=== 解决方案 ==="
echo

echo "方案1: Chrome强制刷新"
echo "1. 打开Chrome开发者工具 (F12)"
echo "2. 右键点击刷新按钮"
echo "3. 选择'清空缓存并硬性重新加载'"
echo "4. 或者快捷键: Ctrl+Shift+R (Windows/Linux) / Cmd+Shift+R (Mac)"
echo

echo "方案2: 清除Chrome缓存"
echo "1. Chrome设置 -> 隐私和安全 -> 清除浏览数据"
echo "2. 时间范围: '所有时间'"
echo "3. 勾选: '缓存的图片和文件'"
echo "4. 点击'清除数据'"
echo

echo "方案3: 无痕模式测试"
echo "1. 打开Chrome无痕窗口 (Ctrl+Shift+N)"
echo "2. 访问 https://token.yishangcloud.cn"
echo "3. 查看是否还有404错误"
echo

echo "方案4: 服务器端强制清除缓存"
echo "在服务器上运行以下命令重启容器:"
echo "docker-compose restart"
echo

echo "=== 验证步骤 ==="
echo "执行上述任一方案后，访问: https://token.yishangcloud.cn"
echo "检查Chrome开发者工具Network标签，确认文件加载状态为200"