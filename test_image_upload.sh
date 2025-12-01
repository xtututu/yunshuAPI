#!/bin/bash

# 测试图片上传功能
echo "测试图片上传功能..."
UPLOAD_RESULT=$(curl -s -X POST -H "Content-Type: application/json" -d '{"image_url": "https://via.placeholder.com/150"}' http://localhost:3000/api/image/upload)

echo "上传结果:"
echo $UPLOAD_RESULT

# 提取返回的图片URL和文件名
IMAGE_URL=$(echo $UPLOAD_RESULT | grep -o '"image_url":"[^"]*"' | sed 's/"image_url":"//' | sed 's/"//')
FILENAME=$(echo $UPLOAD_RESULT | grep -o '"filename":"[^"]*"' | sed 's/"filename":"//' | sed 's/"//')

echo "提取的图片URL: $IMAGE_URL"
echo "提取的文件名: $FILENAME"

# 测试图片访问功能
echo "\n测试图片访问功能..."
if [ ! -z "$FILENAME" ]; then
    echo "使用curl访问图片并检查响应头中的Content-Length..."
    curl -I http://localhost:3000/image/$FILENAME
fi

echo "\n测试完成!"
