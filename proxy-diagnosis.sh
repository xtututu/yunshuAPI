#!/bin/bash

echo "=== 反向代理诊断脚本 ==="
echo ""

# 1. 检查nginx配置（如果存在）
echo "1. 检查Nginx配置："
if command -v nginx &> /dev/null; then
    echo "✅ Nginx已安装"
    nginx -t 2>/dev/null && echo "✅ Nginx配置语法正确" || echo "❌ Nginx配置有错误"
    
    # 查找nginx配置文件
    NGINX_CONF=$(nginx -T 2>/dev/null | grep -E "conf\.conf|nginx\.conf" | head -1 | cut -d' ' -f2)
    if [ -n "$NGINX_CONF" ]; then
        echo "📄 Nginx主配置文件: $NGINX_CONF"
        if [ -f "$NGINX_CONF" ]; then
            echo "📋 Nginx配置内容（相关部分）："
            grep -n -A5 -B5 "location.*assets\|proxy_pass\|upstream" "$NGINX_CONF" 2>/dev/null || echo "未找到相关配置"
        fi
    fi
else
    echo "❌ Nginx未安装"
fi

echo ""

# 2. 检查其他可能的代理配置
echo "2. 检查其他代理配置："
if [ -f "/etc/caddy/Caddyfile" ]; then
    echo "📄 发现Caddy配置："
    cat /etc/caddy/Caddyfile
fi

if [ -f "/etc/apache2/sites-available/*.conf" ]; then
    echo "📄 发现Apache配置："
    ls -la /etc/apache2/sites-available/
fi

echo ""

# 3. 检查端口映射
echo "3. 检查端口映射："
netstat -tlnp | grep :3000 2>/dev/null || echo "未找到3000端口监听"
netstat -tlnp | grep :80 2>/dev/null || echo "未找到80端口监听"
netstat -tlnp | grep :443 2>/dev/null || echo "未找到443端口监听"

echo ""

# 4. 检查Docker端口映射
echo "4. 检查Docker端口映射："
docker ps --format "table {{.Names}}\t{{.Ports}}" | grep -E "new-api|yishangyun" || echo "未找到相关容器"

echo ""

# 5. 测试内部和外部访问
echo "5. 测试访问对比："
echo "内部访问（localhost:3000）："
curl -s -o /dev/null -w "状态码: %{http_code}, 大小: %{size_download} bytes\n" http://localhost:3000/assets/react-core-DxT2a86c.js 2>/dev/null || echo "内部访问失败"

echo ""
echo "外部访问（通过域名）："
curl -s -o /dev/null -w "状态码: %{http_code}, 大小: %{size_download} bytes\n" https://token.yishangcloud.cn/assets/react-core-DxT2a86c.js 2>/dev/null || echo "外部访问失败"

echo ""

# 6. 检查DNS解析
echo "6. DNS解析检查："
nslookup token.yishangcloud.cn 2>/dev/null || echo "DNS解析失败"

echo ""

# 7. 检查防火墙
echo "7. 防火墙状态："
if command -v ufw &> /dev/null; then
    ufw status 2>/dev/null || echo "UFW状态检查失败"
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --state 2>/dev/null || echo "Firewalld状态检查失败"
fi

echo ""
echo "=== 诊断完成 ==="