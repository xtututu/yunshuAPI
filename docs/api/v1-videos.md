# /v1/videos 接口文档

## 概述

`/v1/videos` 接口是一个 OpenAI 兼容的视频生成API，支持文生视频和图生视频功能。该接口采用异步任务模式，提交请求后返回任务ID，可通过任务ID查询生成状态和结果。

## 基础信息

- **接口路径**: `POST /v1/videos`
- **接口路径**: `GET /v1/videos/{task_id}`
- **认证方式**: Bearer Token (需要在请求头中添加 `Authorization: Bearer sk-xxxx`)
- **内容类型**: `application/json`
- **响应格式**: `application/json`


---

## POST /v1/videos - 提交视频生成任务

### 请求参数

| 参数名 | 类型 | 必填 | 默认值 | 示例值 | 描述 |
|--------|------|------|--------|--------|------|
| model | string | 是 | - | "kling-v1" | 模型/样式ID |
| prompt | string | 文生视频时必填 | - | "宇航员站起身走了" | 文本提示词 |
| image | string | 图生视频时必填 | - | "https://example.com/image.jpg" | 图片输入（支持URL或Base64） |
| duration | float64 | 否 | 5.0 | 5.0 | 视频时长（秒） |
| width | int | 否 | 512 | 512 | 视频宽度 |
| height | int | 否 | 512 | 512 | 视频高度 |
| fps | int | 否 | 30 | 30 | 视频帧率 |
| seed | int | 否 | 随机 | 20231234 | 随机种子 |
| n | int | 否 | 1 | 1 | 生成视频数量 |
| response_format | string | 否 | "url" | "url" | 响应格式 |
| user | string | 否 | - | "user-1234" | 用户标识符 |
| metadata | object | 否 | - | - | 厂商特定/自定义参数 |

### metadata 参数说明

`metadata` 字段支持传递厂商特定的参数，例如：

```json
{
  "negative_prompt": "blurry, low quality",
  "style": "cinematic",
  "quality_level": "high",
  "aspect_ratio": "16:9",
  "cfg_scale": 0.7,
  "mode": "std",
  "callback_url": "https://your.domain/callback",
  "external_task_id": "custom-task-001"
}
```

### 请求示例

#### 文生视频请求
```bash
curl -X POST "https://your-domain.com/v1/videos" \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v1",
    "prompt": "宇航员站起身走了",
    "duration": 5.0,
    "width": 512,
    "height": 512,
    "metadata": {
      "negative_prompt": "blurry, low quality",
      "style": "cinematic"
    }
  }'
```

#### 图生视频请求
```bash
curl -X POST "https://your-domain.com/v1/videos" \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v1",
    "image": "https://example.com/input-image.jpg",
    "prompt": "让图片中的人物开始行走",
    "duration": 5.0,
    "width": 512,
    "height": 512
  }'
```

### 响应格式

#### 成功响应 (200 OK)

```json
{
  "task_id": "abcd1234efgh5678",
  "status": "queued"
}
```

#### 错误响应

**400 Bad Request - 参数错误**
```json
{
  "error": {
    "message": "prompt is required for text-to-video generation",
    "type": "invalid_request_error",
    "param": "prompt",
    "code": null
  }
}
```

**401 Unauthorized - 认证失败**
```json
{
  "error": {
    "message": "Invalid API key",
    "type": "invalid_request_error",
    "param": null,
    "code": "invalid_api_key"
  }
}
```

**403 Forbidden - 权限不足**
```json
{
  "error": {
    "message": "Insufficient quota",
    "type": "insufficient_quota",
    "param": null,
    "code": "insufficient_quota"
  }
}
```

**500 Internal Server Error - 服务器错误**
```json
{
  "error": {
    "message": "Internal server error",
    "type": "internal_server_error",
    "param": null,
    "code": null
  }
}
```

---

## GET /v1/videos/{task_id} - 查询视频生成任务

### 请求参数

| 参数名 | 类型 | 位置 | 必填 | 描述 |
|--------|------|------|------|------|
| task_id | string | path | 是 | 任务ID |

### 请求示例

```bash
curl -X GET "https://your-domain.com/v1/videos/abcd1234efgh5678" \
  -H "Authorization: Bearer sk-your-api-key"
```

### 响应格式

#### 任务进行中 (200 OK)

```json
{
  "task_id": "abcd1234efgh5678",
  "status": "in_progress",
  "progress": 45,
  "created_at": 1700000000,
  "metadata": {
    "duration": 5.0,
    "fps": 30,
    "width": 512,
    "height": 512,
    "seed": 20231234
  }
}
```

#### 任务完成 (200 OK)

```json
{
  "task_id": "abcd1234efgh5678",
  "status": "completed",
  "url": "https://cdn.example.com/videos/abcd1234efgh5678.mp4",
  "format": "mp4",
  "created_at": 1700000000,
  "completed_at": 1700000300,
  "expires_at": 1700644800,
  "seconds": "5.0",
  "size": "2.5MB",
  "metadata": {
    "duration": 5.0,
    "fps": 30,
    "width": 512,
    "height": 512,
    "seed": 20231234
  }
}
```

#### 任务失败 (200 OK)

```json
{
  "task_id": "abcd1234efgh5678",
  "status": "failed",
  "created_at": 1700000000,
  "error": {
    "message": "Content policy violation",
    "code": "content_policy_violation"
  },
  "metadata": {
    "duration": 5.0,
    "fps": 30,
    "width": 512,
    "height": 512,
    "seed": 20231234
  }
}
```

### 状态码说明

| 状态码 | 描述 |
|--------|------|
| `queued` | 任务已排队，等待处理 |
| `in_progress` | 任务正在处理中 |
| `completed` | 任务完成，视频已生成 |
| `failed` | 任务失败 |
| `unknown` | 未知状态 |

---

## 错误码说明

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足或配额不足 |
| 404 | 任务不存在 |
| 429 | 请求频率过高 |
| 500 | 服务器内部错误 |

### 业务错误码

| 错误码 | 说明 |
|--------|------|
| `invalid_api_key` | API密钥无效 |
| `insufficient_quota` | 配额不足 |
| `content_policy_violation` | 违反内容政策 |
| `model_not_supported` | 不支持的模型 |
| `invalid_image_format` | 无效的图片格式 |
| `prompt_too_long` | 提示词过长 |
| `rate_limit_exceeded` | 请求频率超限 |

---

## 使用限制

### 请求限制

- **提示词长度**: 最大 2000 字符
- **图片大小**: 最大 10MB
- **支持图片格式**: JPEG, PNG, WebP
- **视频时长**: 1-30 秒
- **视频分辨率**: 最大 1024x1024
- **并发任务数**: 根据用户等级限制

### 配额说明

- 每次视频生成消耗配额根据模型和时长计算
- 失败的任务不会扣除配额
- 配额不足时返回 403 错误

---

## SDK 示例

### JavaScript/Node.js

```javascript
// 提交视频生成任务
async function submitVideoTask() {
  const response = await fetch('https://your-domain.com/v1/videos', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer sk-your-api-key',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model: 'kling-v1',
      prompt: '宇航员站起身走了',
      duration: 5.0,
      width: 512,
      height: 512
    })
  });
  
  const result = await response.json();
  return result.task_id;
}

// 查询任务状态
async function getVideoStatus(taskId) {
  const response = await fetch(`https://your-domain.com/v1/videos/${taskId}`, {
    headers: {
      'Authorization': 'Bearer sk-your-api-key'
    }
  });
  
  return await response.json();
}

// 完整使用示例
async function generateVideo() {
  try {
    // 1. 提交任务
    const taskId = await submitVideoTask();
    console.log('任务ID:', taskId);
    
    // 2. 轮询查询状态
    let status = 'queued';
    while (status === 'queued' || status === 'in_progress') {
      await new Promise(resolve => setTimeout(resolve, 2000)); // 等待2秒
      const result = await getVideoStatus(taskId);
      status = result.status;
      console.log('任务状态:', status, '进度:', result.progress || 0);
      
      if (status === 'completed') {
        console.log('视频生成完成:', result.url);
        return result.url;
      } else if (status === 'failed') {
        console.error('视频生成失败:', result.error.message);
        throw new Error(result.error.message);
      }
    }
  } catch (error) {
    console.error('生成视频时出错:', error);
  }
}

generateVideo();
```

### Python

```python
import requests
import time

class VideoGenerator:
    def __init__(self, api_key, base_url="https://your-domain.com"):
        self.api_key = api_key
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        }
    
    def submit_task(self, model, prompt, **kwargs):
        """提交视频生成任务"""
        url = f"{self.base_url}/v1/videos"
        data = {
            "model": model,
            "prompt": prompt,
            **kwargs
        }
        
        response = requests.post(url, json=data, headers=self.headers)
        response.raise_for_status()
        return response.json()['task_id']
    
    def get_status(self, task_id):
        """查询任务状态"""
        url = f"{self.base_url}/v1/videos/{task_id}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()
    
    def generate_video(self, model, prompt, **kwargs):
        """生成视频（完整流程）"""
        # 提交任务
        task_id = self.submit_task(model, prompt, **kwargs)
        print(f"任务已提交，任务ID: {task_id}")
        
        # 轮询查询状态
        while True:
            result = self.get_status(task_id)
            status = result['status']
            progress = result.get('progress', 0)
            print(f"任务状态: {status}, 进度: {progress}%")
            
            if status == 'completed':
                print(f"视频生成完成: {result['url']}")
                return result['url']
            elif status == 'failed':
                error_msg = result.get('error', {}).get('message', '未知错误')
                raise Exception(f"视频生成失败: {error_msg}")
            
            time.sleep(2)  # 等待2秒后再次查询

# 使用示例
if __name__ == "__main__":
    generator = VideoGenerator("sk-your-api-key")
    
    try:
        video_url = generator.generate_video(
            model="kling-v1",
            prompt="宇航员站起身走了",
            duration=5.0,
            width=512,
            height=512
        )
        print(f"生成的视频URL: {video_url}")
    except Exception as e:
        print(f"生成视频失败: {e}")
```

---

## 最佳实践

### 1. 错误处理

- 始终检查HTTP状态码和响应中的错误信息
- 实现重试机制处理临时性错误
- 设置合理的超时时间

### 2. 性能优化

- 使用适当的轮询间隔（建议2-5秒）
- 实现客户端缓存避免重复查询
- 合理设置并发任务数量

### 3. 安全建议

- 妥善保管API密钥，不要在客户端代码中暴露
- 使用HTTPS进行所有API调用
- 实现请求签名验证（如需要）

### 4. 监控建议

- 记录任务提交和完成时间
- 监控成功率失败率
- 设置告警机制

---

## 更新日志

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持文生视频和图生视频
- 兼容OpenAI API格式
- 支持Kling和Jimeng服务提供商

---

## 技术支持

如有问题或建议，请联系：
- 邮箱: support@example.com
- 文档: https://docs.example.com
- GitHub: https://github.com/example/video-api