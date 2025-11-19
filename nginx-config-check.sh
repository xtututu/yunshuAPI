#!/bin/bash

echo "=== Nginx配置详细检查 ==="
echo ""

# 1. 查找nginx配置文件
echo "1. 查找Nginx配置文件："
find /etc/nginx -name "*.conf" -type f 2>/dev/null | head -10

echo ""

# 2. 检查主配置文件
echo "2. 主配置文件内容："
if [ -f "/etc/nginx/nginx.conf" ]; then
    echo "📄 /etc/nginx/nginx.conf:"
    grep -n -E "include|server|location|proxy_pass" /etc/nginx/nginx.conf 2>/dev/null || echo "未找到相关配置"
fi

echo ""

# 3. 检查sites-available和sites-enabled
echo "3. 检查虚拟主机配置："
for dir in /etc/nginx/sites-available /etc/nginx/sites-enabled /etc/nginx/conf.d; do
    if [ -d "$dir" ]; then
        echo "📁 $dir 目录："
        ls -la "$dir" 2>/dev/null
        echo ""
    fi
done

# 4. 查找包含域名的配置文件
echo "4. 查找包含域名的配置："
find /etc/nginx -name "*.conf" -exec grep -l "token.yishangcloud.cn\|yishangcloud" {} \; 2>/dev/null

echo ""

# 5. 检查具体的server配置
echo "5. 检查server配置块："
for conf in $(find /etc/nginx -name "*.conf" -type f 2>/dev/null); do
    if grep -q "server_name.*token.yishangcloud.cn\|server_name.*yishangcloud" "$conf" 2>/dev/null; then
        echo "📄 找到相关配置文件: $conf"
        echo "配置内容："
        grep -n -A10 -B5 "server_name.*token.yishangcloud.cn\|server_name.*yishangcloud" "$conf" 2>/dev/null
        echo ""
    fi
done

# 6. 检查所有server配置
echo "6. 所有server配置："
find /etc/nginx -name "*.conf" -exec grep -H -n -A15 "server {" {} \; 2>/dev/null | grep -E "server|listen|server_name|location|proxy_pass"

echo ""
echo "=== 检查完成 ==="