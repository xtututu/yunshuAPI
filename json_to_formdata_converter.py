#!/usr/bin/env python3
"""
JSON 到 Form-Data 格式转换工具
用于将 /images/edits 接口的 JSON 参数转换为 multipart/form-data 格式
"""

import json

def convert_json_to_formdata(json_data):
    """
    将 JSON 数据转换为 multipart/form-data 格式
    
    Args:
        json_data (dict): 包含图像编辑参数的 JSON 对象
        
    Returns:
        dict: 包含 form-data 字段的字典
    """
    form_data = {}
    
    # 基本字段转换
    if 'size' in json_data:
        form_data['size'] = json_data['size']
    
    if 'model' in json_data:
        form_data['model'] = json_data['model']
    
    if 'prompt' in json_data:
        form_data['prompt'] = json_data['prompt']
    
    if 'seconds' in json_data:
        form_data['seconds'] = json_data['seconds']
    
    # 处理 input_reference 字段
    if 'input_reference' in json_data:
        input_ref = json_data['input_reference']
        
        # 检查是否是空数组
        if input_ref == []:
            # 如果是空数组，跳过不处理
            pass
        elif isinstance(input_ref, str):
            input_ref = input_ref.strip()
            
            # 检查是否是 URL
            if input_ref.startswith(('http://', 'https://')):
                # 如果是 URL，直接使用
                form_data['input_reference'] = input_ref
            else:
                # 如果是文件路径或 base64 数据，需要特殊处理
                # 这里假设是 URL，但需要清理格式
                form_data['input_reference'] = input_ref.strip('`').strip()
        else:
            # 其他类型的数据，转换为字符串
            form_data['input_reference'] = str(input_ref)
    
    return form_data

def main():
    """主函数 - 示例用法"""
    
    # 示例 JSON 数据 - 正常情况
    example_json = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "input_reference": "https://internal-api-drive-stream.feishu.cn/space/api/box/stream/download/authcode/?code=Y2RmNTQ0ZTkwNjNhZTM2OGIzOWQ1OGI3NmExYTUyOGNfYzdkN2FkMDZjYzhhNWQyYmYwYTI1NDBlMjczNzFlNDFfSUQ6NzU3MjQ5NjI0NDU3NzU1MDMzOF8xNzYzMTA5NDEwOjE3NjMxMTAwMTBfVjM"
    }
    
    # 示例 JSON 数据 - input_reference 为空数组的情况
    example_json_empty_array = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "input_reference": []
    }
    
    print("=== 测试用例 1: 正常情况 ===")
    print("原始 JSON 数据:")
    print(json.dumps(example_json, indent=2, ensure_ascii=False))
    print("\n" + "="*80 + "\n")
    
    # 转换为 form-data
    form_data = convert_json_to_formdata(example_json)
    
    print("转换后的 Form-Data 字段:")
    for key, value in form_data.items():
        print(f"{key}: {value}")
    
    print("\n" + "="*80 + "\n")
    
    # 生成 curl 命令
    print("对应的 curl 命令:")
    curl_cmd = "curl -X POST 'http://localhost:3000/v1/images/edits'"
    for key, value in form_data.items():
        if value:
            curl_cmd += f" -F '{key}={value}'"
    print(curl_cmd)
    
    print("\n" + "="*80 + "\n")
    
    # 生成 Python requests 代码
    print("Python requests 代码示例:")
    print("""
import requests

url = "http://localhost:3000/v1/images/edits"

# Form-data 参数
files = {}
data = {
    "size": "1024x1792",
    "model": "sora2-hd", 
    "seconds": "8",
    "prompt": "动起来",
    "input_reference": "https://internal-api-drive-stream.feishu.cn/space/api/box/stream/download/authcode/?code=Y2RmNTQ0ZTkwNjNhZTM2OGIzOWQ1OGI3NmExYTUyOGNfYzdkN2FkMDZjYzhhNWQyYmYwYTI1NDBlMjczNzFlNDFfSUQ6NzU3MjQ5NjI0NDU3NzU1MDMzOF8xNzYzMTA5NDEwOjE3NjMxMTAwMTBfVjM"
}

# 发送请求
response = requests.post(url, files=files, data=data)
print(f"状态码: {response.status_code}")
print(f"响应内容: {response.text}")
""")
    
    print("\n" + "="*80 + "\n")
    print("=== 测试用例 2: input_reference 为空数组的情况 ===")
    print("原始 JSON 数据:")
    print(json.dumps(example_json_empty_array, indent=2, ensure_ascii=False))
    print("\n" + "="*80 + "\n")
    
    # 转换为 form-data
    form_data_empty = convert_json_to_formdata(example_json_empty_array)
    
    print("转换后的 Form-Data 字段:")
    for key, value in form_data_empty.items():
        print(f"{key}: {value}")
    
    print("\n" + "="*80 + "\n")
    
    # 生成 curl 命令
    print("对应的 curl 命令:")
    curl_cmd_empty = "curl -X POST 'http://localhost:3000/v1/images/edits'"
    for key, value in form_data_empty.items():
        if value:
            curl_cmd_empty += f" -F '{key}={value}'"
    print(curl_cmd_empty)
    
    print("\n" + "="*80 + "\n")
    
    # 生成 Python requests 代码
    print("Python requests 代码示例:")
    print("""
import requests

url = "http://localhost:3000/v1/images/edits"

# Form-data 参数 (input_reference 为空数组时)
files = {}
data = {
    "size": "1024x1792",
    "model": "sora2-hd", 
    "seconds": "8",
    "prompt": "动起来"
    # input_reference 为空数组，不包含该字段
}

# 发送请求
response = requests.post(url, files=files, data=data)
print(f"状态码: {response.status_code}")
print(f"响应内容: {response.text}")
""")

if __name__ == "__main__":
    main()