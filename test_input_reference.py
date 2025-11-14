#!/usr/bin/env python3
"""
测试 /v1/images/edits 接口的 input_reference 参数支持
"""

import requests
import json
import os

def test_json_with_input_reference():
    """测试 JSON 请求格式的 input_reference 支持"""
    print("=" * 60)
    print("测试: JSON 请求格式的 input_reference 支持")
    
    # 测试用例1: 使用 image 字段
    print("\n--- 测试用例1: 使用 image 字段 ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "image": {"url": "https://example.com/image1.jpg"}
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    # 测试用例2: 使用 input_reference 字段
    print("\n--- 测试用例2: 使用 input_reference 字段 ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "input_reference": {"url": "https://example.com/input_ref.jpg"}
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    # 测试用例3: 同时使用 image 和 input_reference 字段
    print("\n--- 测试用例3: 同时使用 image 和 input_reference 字段 ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "image": {"url": "https://example.com/image2.jpg"},
        "input_reference": {"url": "https://example.com/input_ref2.jpg"}
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    # 测试用例4: 字符串类型的 input_reference
    print("\n--- 测试用例4: 字符串类型的 input_reference ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "input_reference": "https://example.com/string_ref.jpg"
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    print("\n✅ JSON 请求格式测试用例定义完成")

def test_form_data_with_input_reference():
    """测试 multipart/form-data 格式的 input_reference 支持"""
    print("\n" + "=" * 60)
    print("测试: multipart/form-data 格式的 input_reference 支持")
    
    # 测试用例1: 使用 image 文件
    print("\n--- 测试用例1: 使用 image 文件 ---")
    print("请求格式: multipart/form-data")
    print("字段: image (文件上传)")
    
    # 测试用例2: 使用 input_reference 文件
    print("\n--- 测试用例2: 使用 input_reference 文件 ---")
    print("请求格式: multipart/form-data")
    print("字段: input_reference (文件上传)")
    
    # 测试用例3: 同时使用 image 和 input_reference 文件
    print("\n--- 测试用例3: 同时使用 image 和 input_reference 文件 ---")
    print("请求格式: multipart/form-data")
    print("字段: image (文件上传), input_reference (文件上传)")
    
    print("\n✅ multipart/form-data 请求格式测试用例定义完成")

def test_error_cases():
    """测试错误情况"""
    print("\n" + "=" * 60)
    print("测试: 错误情况")
    
    # 测试用例1: 没有附件
    print("\n--- 测试用例1: 没有附件 (应该报错) ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来"
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    print("预期结果: 应该返回错误 'at least one attachment (image or input_reference) is required'")
    
    # 测试用例2: input_reference 为空
    print("\n--- 测试用例2: input_reference 为空 ---")
    data = {
        "size": "1024x1792",
        "model": "sora2-hd",
        "seconds": "8",
        "prompt": "动起来",
        "input_reference": None
    }
    
    print(f"请求数据: {json.dumps(data, indent=2, ensure_ascii=False)}")
    
    print("\n✅ 错误情况测试用例定义完成")

def main():
    print("开始测试 /v1/images/edits 接口的 input_reference 参数支持")
    print("=" * 60)
    
    # 测试 JSON 请求格式
    test_json_with_input_reference()
    
    # 测试 multipart/form-data 格式
    test_form_data_with_input_reference()
    
    # 测试错误情况
    test_error_cases()
    
    print("\n" + "=" * 60)
    print("测试用例定义完成")
    print("\n说明:")
    print("1. 现在 /v1/images/edits 接口支持两种附件上传方式:")
    print("   - image 字段: 传统的图像附件")
    print("   - input_reference 字段: 新的参考图像附件")
    print("2. 两种方式可以同时使用，但至少需要提供一种")
    print("3. 支持 JSON 和 multipart/form-data 两种请求格式")
    print("4. JSON 格式支持 URL 类型的附件")
    print("5. multipart/form-data 格式支持文件上传")
    
    print("\n✅ 所有测试用例已定义，可以开始实际测试")

if __name__ == "__main__":
    main()