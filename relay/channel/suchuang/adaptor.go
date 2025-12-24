package suchuang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"xunkecloudAPI/common"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/logger"
	relaycommon "xunkecloudAPI/relay/common"
	"xunkecloudAPI/types"

	"github.com/gin-gonic/gin"
)

// ChannelName 渠道名称
const ChannelName = "suchuang"

// ModelList 模型列表
var ModelList = []string{
	"nano-banana-pro_4k",
}

// SuchuangAdaptor 速创渠道适配器
// 用于处理速创API的请求和响应适配
// 实现了channel.Adaptor接口
// 目前仅支持图片生成功能，其他功能返回错误
// 图片生成需要创建任务并轮询结果
// 创建任务接口：/api/img/nanoBanana-pro
// 轮询结果接口：/api/img/drawDetail
// 响应需要转换为OpenAI格式
// 所有请求头中的Authorization不需要Bearer拼接
type SuchuangAdaptor struct {
	channelType int
	baseURL     string
	apiKey      string
}

// Adaptor 实现channel.Adaptor接口的适配器类型
type Adaptor struct {
	SuchuangAdaptor
}

// NewSuchuangAdaptor 创建速创渠道适配器实例
// 接收渠道类型、基础URL和API密钥作为参数
// 返回SuchuangAdaptor实例
func NewSuchuangAdaptor(channelType int, baseURL, apiKey string) *SuchuangAdaptor {
	return &SuchuangAdaptor{
		channelType: channelType,
		baseURL:     baseURL,
		apiKey:      apiKey,
	}
}

// Init 初始化适配器
// 实现channel.Adaptor接口的Init方法
// 接收relaycommon.RelayInfo参数
// 从参数中获取渠道类型、基础URL和API密钥
func (s *SuchuangAdaptor) Init(info *relaycommon.RelayInfo) {
	s.channelType = info.ChannelType
	s.baseURL = info.ChannelBaseUrl
	s.apiKey = info.ApiKey
}

// GetRequestURL 获取请求URL
// 实现channel.Adaptor接口的GetRequestURL方法
// 接收relaycommon.RelayInfo参数
// 根据请求路径和渠道类型构建完整URL
// 目前仅支持图片生成相关接口
func (s *SuchuangAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 注意：这里没有可用的context，因为该方法不接收gin.Context参数
	// 检查DEBUG模式是否启用
	if common.DebugEnabled {
		// 直接输出到控制台以确保我们能看到日志
		fmt.Println("[SUCHUANG] DEBUG MODE IS ENABLED")
		fmt.Printf("[SUCHUANG] GetRequestURL called with: baseURL=%s, RequestURLPath=%s\n", s.baseURL, info.RequestURLPath)
	}
	// 如果是图片生成请求，返回特定的API端点
	if info.RequestURLPath == "/v1/images/generations" {
		url := fmt.Sprintf("%s/api/img/nanoBanana-pro", s.baseURL)
		if common.DebugEnabled {
			fmt.Printf("[SUCHUANG] Generated image generation URL: %s\n", url)
		}
		return url, nil
	}
	// 如果是轮询请求，返回轮询API端点
	if info.RequestURLPath == "/api/img/drawDetail" {
		url := fmt.Sprintf("%s/api/img/drawDetail", s.baseURL)
		if common.DebugEnabled {
			fmt.Printf("[SUCHUANG] Generated poll URL: %s\n", url)
		}
		return url, nil
	}
	// 其他请求路径返回错误
	if common.DebugEnabled {
		fmt.Printf("[SUCHUANG] Unsupported request URL path: %s\n", info.RequestURLPath)
	}
	return "", errors.New("unsupported request URL path for suchuang channel")
}

// SetupRequestHeader 设置请求头
// 实现channel.Adaptor接口的SetupRequestHeader方法
// 接收gin.Context、http.Header和relaycommon.RelayInfo参数
// 设置Content-Type和Authorization头
// Authorization头不需要Bearer拼接
func (s *SuchuangAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	// 设置Content-Type为application/json
	req.Set("Content-Type", "application/json")
	// 设置Authorization头，不需要Bearer拼接
	req.Set("Authorization", s.apiKey)
	return nil
}

// ConvertOpenAIRequest 转换OpenAI请求到速创API请求
// 实现channel.Adaptor接口的ConvertOpenAIRequest方法
// 接收gin.Context、relaycommon.RelayInfo和*dto.GeneralOpenAIRequest参数
// 目前仅支持图片生成请求转换
func (s *SuchuangAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	// 仅支持图片生成请求
	if info.RequestURLPath != "/v1/images/generations" {
		return nil, errors.New("only image generation requests are supported for suchuang channel")
	}

	// 从GeneralOpenAIRequest创建ImageRequest
	imageRequest := &dto.ImageRequest{
		Model: request.Model,
	}

	// 处理Prompt字段
	if request.Prompt != nil {
		if promptStr, ok := request.Prompt.(string); ok {
			imageRequest.Prompt = promptStr
		} else if promptBytes, ok := request.Prompt.([]byte); ok {
			imageRequest.Prompt = string(promptBytes)
		}
	}

	// 对于ConvertOpenAIRequest，image字段可能不在request中
	// 系统应该会为图片请求直接调用ConvertImageRequest方法

	return s.ConvertImageRequest(c, info, *imageRequest)
}

// ConvertRerankRequest 转换重排请求
// 实现channel.Adaptor接口的ConvertRerankRequest方法
// 目前不支持重排请求，直接返回错误
func (s *SuchuangAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("Rerank not implemented for suchuang channel")
}

// ConvertEmbeddingRequest 转换嵌入请求
// 实现channel.Adaptor接口的ConvertEmbeddingRequest方法
// 目前不支持嵌入请求，直接返回错误
func (s *SuchuangAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("Embedding not implemented for suchuang channel")
}

// ConvertAudioRequest 转换音频请求
// 实现channel.Adaptor接口的ConvertAudioRequest方法
// 目前不支持音频请求，直接返回错误
func (s *SuchuangAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("Audio not implemented for suchuang channel")
}

// ConvertImageRequest 转换图片生成请求
// 实现channel.Adaptor接口的ConvertImageRequest方法
// 接收gin.Context、relaycommon.RelayInfo和dto.ImageRequest参数
// 转换OpenAI格式的图片请求到速创API格式
func (s *SuchuangAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[SUCHUANG] ConvertImageRequest called with: Model=%s, Prompt=%s", request.Model, request.Prompt)
	// 解析image数组
	var imageUrls []string
	if request.Image != nil {
		logger.LogDebug(ctx, "[SUCHUANG] Raw image data: %s", string(request.Image))
		var images []string
		if err := json.Unmarshal(request.Image, &images); err != nil {
			// 如果解析失败，尝试解析为单个字符串
			var singleImage string
			if err2 := json.Unmarshal(request.Image, &singleImage); err2 == nil {
				imageUrls = append(imageUrls, singleImage)
				logger.LogDebug(ctx, "[SUCHUANG] Parsed single image URL: %s", singleImage)
			} else {
				logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Failed to parse image data: %v", err))
				return nil, err
			}
		} else {
			imageUrls = images
			logger.LogDebug(ctx, "[SUCHUANG] Parsed image URLs: %v", images)
		}
	}

	// 构建速创API请求体
	suchuangRequest := map[string]interface{}{
		"prompt":      request.Prompt,
		"aspectRatio": "auto", // 固定值
		"imageSize":   "4K",   // 固定值
		"img_url":     imageUrls,
	}

	requestData, _ := json.Marshal(suchuangRequest)
	logger.LogDebug(ctx, "[SUCHUANG] Generated request body: %s", string(requestData))
	return suchuangRequest, nil
}

// ConvertOpenAIResponsesRequest 转换OpenAI响应请求
// 实现channel.Adaptor接口的ConvertOpenAIResponsesRequest方法
// 目前不支持此功能，直接返回错误
func (s *SuchuangAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("OpenAIResponses not implemented for suchuang channel")
}

// DoRequest 执行请求
// 实现channel.Adaptor接口的DoRequest方法
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 执行HTTP请求并返回响应
// 如果是图片生成请求，需要创建任务并轮询结果
func (s *SuchuangAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[SUCHUANG] DoRequest called with RequestURLPath: %s", info.RequestURLPath)

	// 读取并记录请求体
	var requestBodyCopy io.Reader
	if requestBody != nil {
		bodyBytes, _ := io.ReadAll(requestBody)
		logger.LogDebug(ctx, "[SUCHUANG] Original request body: %s", string(bodyBytes))
		requestBodyCopy = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// 如果是图片生成请求，需要特殊处理
	if info.RequestURLPath == "/v1/images/generations" {
		// 创建图片生成任务
		taskID, err := s.createImageTask(c, info, requestBodyCopy)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] createImageTask failed: %v", err))
			return nil, err
		}

		// 轮询获取任务结果
		responseData, err := s.pollImageResult(c, info, taskID)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] pollImageResult failed: %v", err))
			return nil, err
		}

		logger.LogDebug(ctx, "[SUCHUANG] Final response: %v", responseData)

		// 直接返回OpenAI格式的响应数据
		return responseData, nil
	}

	// 其他请求使用GetRequestURL获取正确的URL
	fullURL, err := s.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] GetRequestURL failed: %v", err))
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBody)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] New request failed: %v", err))
		return nil, err
	}

	// 设置请求头
	if err := s.SetupRequestHeader(c, &req.Header, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] SetupRequestHeader failed: %v", err))
		return nil, err
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Client Do failed: %v", err))
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Read response body failed: %v", err))
		return nil, err
	}

	// 解析响应
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// 如果解析失败，返回原始字符串
		logger.LogDebug(ctx, "[SUCHUANG] Unmarshal response failed, returning raw string: %s", string(body))
		return string(body), nil
	}

	logger.LogDebug(ctx, "[SUCHUANG] Response parsed successfully: %v", result)
	return result, nil
}

// DoResponse 处理响应
// 实现channel.Adaptor接口的DoResponse方法
// 接收gin.Context、http.Response和relaycommon.RelayInfo参数
// 处理响应并转换为OpenAI格式
// 目前仅支持图片生成响应处理
func (s *SuchuangAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	ctx := c.Request.Context()

	// 如果响应为空，直接返回
	if resp == nil {
		logger.LogDebug(ctx, "[SUCHUANG] Response is nil, returning empty usage")
		return &dto.Usage{}, nil
	}

	// 读取响应体
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(errors.New("Failed to read response body"), types.ErrorCodeBadResponse, 500)
	}

	// 如果响应体为空，直接返回
	if len(body) == 0 {
		logger.LogDebug(ctx, "[SUCHUANG] Response body is empty, returning empty usage")
		return &dto.Usage{}, nil
	}

	logger.LogDebug(ctx, "[SUCHUANG] Response body: %s", string(body))

	// 重置响应体，以便后续处理
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	// 检查是否已经是OpenAI格式的响应（有data字段）
	var openAIRespCheck struct {
		Data interface{} `json:"data"`
	}
	if json.Unmarshal(body, &openAIRespCheck) == nil && openAIRespCheck.Data != nil {
		// 如果已经是OpenAI格式的响应，直接返回
		logger.LogDebug(ctx, "[SUCHUANG] Response is already in OpenAI format, skipping processing")
		return &dto.Usage{}, nil
	}

	// 根据请求路径处理响应
	switch info.RequestURLPath {
	case "/v1/images/generations":
		// 处理图片生成任务创建响应
		var createTaskResp struct {
			Msg  string `json:"msg"`
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
			Code     int     `json:"code"`
			ExecTime float64 `json:"exec_time,omitempty"`
			IP       string  `json:"ip,omitempty"`
		}

		if err := json.Unmarshal(body, &createTaskResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] DoResponse failed to parse response: %v, body: %s", err, string(body)))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
		}

		if createTaskResp.Code != 200 {
			return nil, types.NewErrorWithStatusCode(errors.New(createTaskResp.Msg), types.ErrorCodeBadResponse, 500)
		}

		// 返回任务ID
		return map[string]int{"task_id": createTaskResp.Data.ID}, nil

	case "/api/img/drawDetail":
		// 处理轮询结果响应
		var pollResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				ID         int    `json:"id"`
				TaskID     string `json:"task_id"`
				Status     int    `json:"status"` // 2表示完成
				Size       string `json:"size"`
				Prompt     string `json:"prompt"`
				ImageURL   string `json:"image_url"`
				FailReason string `json:"fail_reason"`
				CreatedAt  string `json:"created_at"`
				UpdatedAt  string `json:"updated_at"`
			} `json:"data"`
			ExecTime float64 `json:"exec_time"`
			IP       string  `json:"ip"`
		}

		if err := json.Unmarshal(body, &pollResp); err != nil {
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse poll response"), types.ErrorCodeBadResponse, 500)
		}

		if pollResp.Code != 200 {
			return nil, types.NewErrorWithStatusCode(errors.New(pollResp.Msg), types.ErrorCodeBadResponse, 500)
		}

		// 检查任务状态
		if pollResp.Data.Status != 2 {
			return nil, types.NewErrorWithStatusCode(errors.New("Task not completed yet"), types.ErrorCodeInvalidRequest, 400)
		}

		// 转换为OpenAI格式响应
		openAIResp := struct {
			Data []struct {
				URL string `json:"url"`
			} `json:"data"`
			Created int64 `json:"created"`
		}{
			Data: []struct {
				URL string `json:"url"`
			}{{
				URL: pollResp.Data.ImageURL,
			}},
			Created: time.Now().Unix(),
		}

		// 将转换后的响应写回响应体
		openAIRespBody, _ := json.Marshal(openAIResp)
		resp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))

		// 返回Usage实例
		return &dto.Usage{}, nil

	default:
		// 其他请求返回Usage实例
		return &dto.Usage{}, nil
	}
}

// GetModelList 获取支持的模型列表
// 实现channel.Adaptor接口的GetModelList方法
// 返回ModelList变量
func (s *SuchuangAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 获取渠道名称
// 实现channel.Adaptor接口的GetChannelName方法
// 返回ChannelName常量
func (s *SuchuangAdaptor) GetChannelName() string {
	return ChannelName
}

// ConvertClaudeRequest 转换Claude请求
// 实现channel.Adaptor接口的ConvertClaudeRequest方法
// 目前不支持Claude请求，直接返回错误
func (s *SuchuangAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("ClaudeRequest not implemented for suchuang channel")
}

// ConvertGeminiRequest 将Gemini请求转换为内部请求格式
// 实现channel.Adaptor接口的ConvertGeminiRequest方法
func (s *SuchuangAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertGeminiProRequest 将Gemini Pro请求转换为内部请求格式
// 实现channel.Adaptor接口的ConvertGeminiProRequest方法
func (s *SuchuangAdaptor) ConvertGeminiProRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertGemini15ProRequest 将Gemini 1.5 Pro请求转换为内部请求格式
// 实现channel.Adaptor接口的ConvertGemini15ProRequest方法
func (s *SuchuangAdaptor) ConvertGemini15ProRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// createImageTask 创建图片生成任务
// 内部方法，用于创建图片生成任务
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 执行HTTP请求并返回任务ID
func (s *SuchuangAdaptor) createImageTask(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (int, error) {
	ctx := c.Request.Context()
	// 创建HTTP客户端
	client := &http.Client{}

	// 使用GetRequestURL获取正确的URL
	url, err := s.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] GetRequestURL failed in createImageTask: %v", err))
		return 0, err
	}
	logger.LogDebug(ctx, "[SUCHUANG] Creating image task with URL: %s", url)

	// 读取请求体内容用于日志
	requestBodyBytes, _ := io.ReadAll(requestBody)
	requestBodyCopy := io.NopCloser(bytes.NewBuffer(requestBodyBytes))
	logger.LogDebug(ctx, "[SUCHUANG] Request body: %s", string(requestBodyBytes))

	req, err := http.NewRequest("POST", url, requestBodyCopy)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Error creating request: %v", err))
		return 0, err
	}

	// 设置请求头
	if err := s.SetupRequestHeader(c, &req.Header, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Error setting up request header: %v", err))
		return 0, err
	}

	// 记录请求头
	logger.LogDebug(ctx, "[SUCHUANG] Request headers: %v", req.Header)

	// 发送请求
	logger.LogDebug(ctx, "[SUCHUANG] Sending request to %s", url)
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Error sending request: %v", err))
		return 0, err
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, _ := io.ReadAll(resp.Body)
	logger.LogDebug(ctx, "[SUCHUANG] Response status: %d", resp.StatusCode)
	logger.LogDebug(ctx, "[SUCHUANG] Response headers: %v", resp.Header)
	logger.LogDebug(ctx, "[SUCHUANG] Response body: %s", string(respBody))

	// 处理响应
	var createTaskResp struct {
		Msg  string `json:"msg"`
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
		Code int `json:"code"`
	}

	if err := json.Unmarshal(respBody, &createTaskResp); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Error unmarshalling response: %v", err))
		return 0, err
	}

	if createTaskResp.Code != 200 {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] API returned error: Code=%d, Msg=%s", createTaskResp.Code, createTaskResp.Msg))
		return 0, errors.New(createTaskResp.Msg)
	}

	logger.LogDebug(ctx, "[SUCHUANG] Image task created successfully with ID: %d", createTaskResp.Data.ID)
	return createTaskResp.Data.ID, nil
}

// pollImageResult 轮询图片生成结果
// 内部方法，用于轮询图片生成结果
// 接收gin.Context、relaycommon.RelayInfo和任务ID参数
// 定期查询任务状态，直到完成或超时
// 返回转换后的OpenAI格式响应
func (s *SuchuangAdaptor) pollImageResult(c *gin.Context, info *relaycommon.RelayInfo, taskID int) (any, error) {
	ctx := c.Request.Context()
	// 创建HTTP客户端
	client := &http.Client{}

	// 轮询参数
	maxAttempts := 60           // 最大尝试次数
	interval := 5 * time.Second // 轮询间隔

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 创建轮询请求
		url := fmt.Sprintf("%s/api/img/drawDetail?id=%d", s.baseURL, taskID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// 设置请求头
		if err := s.SetupRequestHeader(c, &req.Header, info); err != nil {
			return nil, err
		}

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		// 读取响应内容
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		// 添加调试日志，查看原始响应内容
		if common.DebugEnabled {
			logger.LogDebug(ctx, "[SUCHUANG] Poll response: %s", string(respBody))
		}

		// 处理响应
		var pollResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				ID         int    `json:"id"`
				TaskID     string `json:"task_id"`
				Status     int    `json:"status"` // 2表示完成
				Size       string `json:"size"`
				Prompt     string `json:"prompt"`
				ImageURL   string `json:"image_url"`
				FailReason string `json:"fail_reason"`
				CreatedAt  string `json:"created_at"`
				UpdatedAt  string `json:"updated_at"`
			} `json:"data"`
			ExecTime float64 `json:"exec_time"`
			IP       string  `json:"ip"`
		}

		if err := json.Unmarshal(respBody, &pollResp); err != nil {
			return nil, err
		}

		if pollResp.Code != 200 {
			return nil, errors.New(pollResp.Msg)
		}

		// 检查任务状态
		if pollResp.Data.Status == 2 {
			// 任务完成，添加详细调试日志
			logger.LogDebug(ctx, "[SUCHUANG] Task completed, ImageURL: %s", pollResp.Data.ImageURL)
			logger.LogDebug(ctx, "[SUCHUANG] Full poll response data: %+v", pollResp.Data)

			// 转换为OpenAI格式响应
			openAIResp := struct {
				Data []struct {
					URL string `json:"url"`
				} `json:"data"`
				Created int64 `json:"created"`
			}{
				Data: []struct {
					URL string `json:"url"`
				}{{
					URL: pollResp.Data.ImageURL,
				}},
				Created: time.Now().Unix(),
			}

			// 记录最终响应
			logger.LogDebug(ctx, "[SUCHUANG] Final OpenAI response: %+v", openAIResp)

			return openAIResp, nil
		}

		// 任务未完成，等待一段时间后继续轮询
		time.Sleep(interval)
	}

	// 轮询超时
	return nil, errors.New("image generation timeout")
}
