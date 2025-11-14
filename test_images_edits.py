#!/usr/bin/env python3
"""
测试 /v1/images/edits 接口的 JSON 请求格式
"""

import requests
import json

# 测试 URL
url = "http://localhost:3000/v1/images/edits"

# 测试用例 1: 正常的 JSON 请求（不带 input_reference 字段）
json_data_1 = {
    "size": "1024x1792",
    "model": "sora2-hd",
    "seconds": "8",
    "prompt": "动起来"
}

# 测试用例 2: 包含 input_reference 字段的 JSON 请求
json_data_2 = {
    "size": "1024x1792",
    "model": "sora2-hd",
    "seconds": "8",
    "prompt": "动起来",
    "input_reference": "https://example.com/image.jpg"
}

# 测试用例 3: input_reference 为空数组的 JSON 请求
json_data_3 = {
    "size": "1024x1792",
    "model": "sora2-hd",
    "seconds": "8",
    "prompt": "动起来",
    "input_reference": []
}

headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer your-api-key-here"  # 替换为实际的 API key
}

def test_request(test_name, data):
    print(f"\n{'='*60}")
    print(f"测试: {test_name}")
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    try:
        response = requests.post(url, json=data, headers=headers)
        print(f"状态码: {response.status_code}")
        print(f"响应内容: {response.text}")
        
        if response.status_code == 200:
            print("✅ 请求成功")
        else:
            print("❌ 请求失败")
            
    except Exception as e:
        print(f"❌ 请求异常: {e}")

if __name__ == "__main__":
    print("开始测试 /v1/images/edits 接口的 JSON 请求格式")
    
    # 测试三个用例
    test_request("正常 JSON 请求（不带 input_reference）", json_data_1)
    test_request("包含 input_reference URL 的 JSON 请求", json_data_2)
    test_request("input_reference 为空数组的 JSON 请求", json_data_3)
    
    print(f"\n{'='*60}")
    print("测试完成")