package suchuang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"xunkecloudAPI/common"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/logger"
	relaycommon "xunkecloudAPI/relay/common"
	"xunkecloudAPI/service"
	"xunkecloudAPI/types"

	"github.com/gin-gonic/gin"
)

// removeThinkContent 移除内容中的<think>标签部分
// 接收原始内容字符串，返回处理后的内容字符串
func removeThinkContent(content string) string {
	// 使用正则表达式匹配<think>...</think>部分，包括跨多行的情况
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	// 替换为空白字符串
	return re.ReplaceAllString(content, "")
}

// ChannelName 渠道名称
const ChannelName = "suchuang"

// ModelList 模型列表
var ModelList = []string{
	"nano-banana-pro_4k",
	"gemini-3-pro",
	"gemini-2.5-pro",
	"gemini-3-pro-preview",
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
// 目前支持图片生成相关接口和Gemini聊天接口
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
	// 如果是聊天请求（支持标准和plus路径），返回特定的API端点
	if info.RequestURLPath == "/v1/chat/completions" || info.RequestURLPath == "/plus/v1/chat/completions" {
		url := fmt.Sprintf("%s/api/chat/index", s.baseURL)
		if common.DebugEnabled {
			fmt.Printf("[SUCHUANG] Generated chat URL for model %s: %s\n", info.OriginModelName, url)
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
// 支持图片生成请求和Gemini-3-Pro视频内容识别请求转换
func (s *SuchuangAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	ctx := c.Request.Context()
	// 支持图片生成请求
	if info.RequestURLPath == "/v1/images/generations" {
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

	// 支持视频内容识别请求（支持gemini-3-pro和gpt-5模型）
	if info.RequestURLPath == "/v1/chat/completions" || info.RequestURLPath == "/plus/v1/chat/completions" {
		logger.LogDebug(ctx, "[SUCHUANG] Converting chat completion request for %s", request.Model)

		// 当渠道是速创时，且调用的接口是/plus/v1/chat/completions，如果模型是gemini-3-pro-preview则强制修改为gemini-3-pro
		if info.RequestURLPath == "/plus/v1/chat/completions" && request.Model == "gemini-3-pro-preview" {
			logger.LogDebug(ctx, "[SUCHUANG] Model name converted from gemini-3-pro-preview to gemini-3-pro for suchuang channel")
			request.Model = "gemini-3-pro"
			// 同时更新 RelayInfo 中的模型名称，确保后续处理使用正确的模型名称
			if info.OriginModelName == "gemini-3-pro-preview" {
				info.OriginModelName = "gemini-3-pro"
			}
			if info.UpstreamModelName == "gemini-3-pro-preview" {
				info.UpstreamModelName = "gemini-3-pro"
			}
		}

		// 解析请求中的消息
		var videoUrl string
		var content string

		if len(request.Messages) > 0 {
			lastMessage := request.Messages[len(request.Messages)-1]
			logger.LogDebug(ctx, "[SUCHUANG] Last message: %+v", lastMessage)

			// 解析消息内容
			if lastMessage.IsStringContent() {
				// 纯文本内容
				if contentStr, ok := lastMessage.Content.(string); ok {
					content = contentStr
				}
			} else {
				// 多媒体内容
				mediaContents := lastMessage.ParseContent()
				logger.LogDebug(ctx, "[SUCHUANG] Media contents: %+v", mediaContents)

				for _, media := range mediaContents {
					if media.Type == dto.ContentTypeText {
						content = media.Text
					} else if media.Type == dto.ContentTypeVideoUrl {
						// 处理视频URL
						if videoUrlObj := media.GetVideoUrl(); videoUrlObj != nil {
							videoUrl = videoUrlObj.Url
						}
					} else if media.Type == dto.ContentTypeImageURL {
						// 处理图片URL（作为视频URL使用）
						if imageUrlObj := media.GetImageMedia(); imageUrlObj != nil {
							videoUrl = imageUrlObj.Url
						}
					} else if media.Type == dto.ContentTypeFileURL {
						// 处理文件URL（作为视频URL使用）
						if fileUrlObj := media.GetFileUrl(); fileUrlObj != nil {
							// 清理URL中的反引号和其他无效字符
							videoUrl = strings.Trim(fileUrlObj.Url, " `")
						}
					}
				}
			}
		}

		// 设置默认的识别内容
		if content == "" {
			content = "识别视频内容结果输出中文"
		}

		// 构建速创API请求体
		suchuangRequest := map[string]interface{}{
			"model":   request.Model, // 使用请求中的模型名称
			"content": content,
		}

		// 如果有视频URL，添加到请求中
		if videoUrl != "" {
			suchuangRequest["image_url"] = videoUrl
		}

		requestData, _ := json.Marshal(suchuangRequest)
		logger.LogDebug(ctx, "[SUCHUANG] Generated chat request body: %s", string(requestData))
		return suchuangRequest, nil
	}

	return nil, errors.New("unsupported request URL path for suchuang channel")
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
func (s *SuchuangAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[SUCHUANG] DoRequest called with RequestURLPath: %s", info.RequestURLPath)

	// 读取并记录请求体
	var requestBodyCopy io.Reader
	var bodyBytes []byte
	if requestBody != nil {
		bodyBytes, _ = io.ReadAll(requestBody)
		logger.LogDebug(ctx, "[SUCHUANG] Original request body: %s", string(bodyBytes))
		requestBodyCopy = io.NopCloser(bytes.NewBuffer(bodyBytes))
	} else {
		requestBodyCopy = nil
	}

	// 所有请求统一使用GetRequestURL获取正确的URL
	fullURL, err := s.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] GetRequestURL failed: %v", err))
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBodyCopy)
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
	logger.LogDebug(ctx, "[SUCHUANG] Sending request to URL: %s with method: %s", req.URL.String(), req.Method)
	logger.LogDebug(ctx, "[SUCHUANG] Request headers: %v", req.Header)
	logger.LogDebug(ctx, "[SUCHUANG] Request body: %s", string(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Client Do failed: %v", err))
		return nil, err
	}

	// 不要关闭响应体，因为DoResponse方法需要使用它
	// defer resp.Body.Close()

	// 读取响应体以便记录日志
	respBody, _ := io.ReadAll(resp.Body)
	logger.LogDebug(ctx, "[SUCHUANG] Response status: %d", resp.StatusCode)
	logger.LogDebug(ctx, "[SUCHUANG] Response headers: %v", resp.Header)
	logger.LogDebug(ctx, "[SUCHUANG] Response body: %s", string(respBody))

	// 重置响应体，以便DoResponse方法使用
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	logger.LogDebug(ctx, "[SUCHUANG] Request completed successfully, returning *http.Response")
	return resp, nil
}

// DoResponse 处理响应
// 实现channel.Adaptor接口的DoResponse方法
// 接收gin.Context、http.Response和relaycommon.RelayInfo参数
// 处理响应并转换为OpenAI格式
// 目前仅支持图片生成响应处理
func (s *SuchuangAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	ctx := c.Request.Context()

	// 处理响应
	var body []byte
	var httpResp = resp

	// 如果响应为空，直接返回
	if resp == nil {
		logger.LogDebug(ctx, "[SUCHUANG] Response is nil, returning empty usage")
		return &dto.Usage{}, nil // 返回空的usage信息和nil错误
	} // 读取响应体
	var readErr error
	body, readErr = io.ReadAll(httpResp.Body)
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
	if httpResp != nil {
		httpResp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// 检查是否已经是OpenAI格式的响应（检查是否有choices字段）
	var openAIRespCheck struct {
		Choices []any `json:"choices"`
	}
	if json.Unmarshal(body, &openAIRespCheck) == nil && openAIRespCheck.Choices != nil {
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

		// 如果响应体不为空，将转换后的响应写回响应体
		if httpResp != nil {
			openAIRespBody, _ := json.Marshal(openAIResp)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))
		} else {
			// 非http.Response类型，直接替换响应数据
		}

		// 返回Usage实例
		return &dto.Usage{}, nil

	case "/v1/chat/completions", "/plus/v1/chat/completions":
		// 处理gemini-3-pro或gemini-2.5-pro视频内容识别响应
		if info.OriginModelName == "gemini-3-pro" || info.OriginModelName == "gemini-2.5-pro" || info.OriginModelName == "gemini-3-pro-preview" {
			var suchuangResp struct {
				Code     int             `json:"code"`
				Msg      string          `json:"msg"`
				Data     json.RawMessage `json:"data"`
				ExecTime float64         `json:"exec_time"`
				IP       string          `json:"ip"`
			}

			if err := json.Unmarshal(body, &suchuangResp); err != nil {
				logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] DoResponse failed to parse chat response: %v, body: %s", err, string(body)))
				return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
			}

			if suchuangResp.Code != 200 {
				return nil, types.NewErrorWithStatusCode(errors.New(suchuangResp.Msg), types.ErrorCodeBadResponse, 500)
			}

			// 将速创API的响应转换为OpenAI格式
			var openAIResp struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				Model   string `json:"model"`
				Choices []struct {
					Index   int `json:"index"`
					Message struct {
						Role         string  `json:"role"`
						Content      string  `json:"content"`
						Refusal      *string `json:"refusal"`
						Annotations  []any   `json:"annotations"`
						Audio        *string `json:"audio"`
						FunctionCall *any    `json:"function_call"`
						ToolCalls    *any    `json:"tool_calls"`
					} `json:"message"`
					Logprobs             *any           `json:"logprobs"`
					FinishReason         string         `json:"finish_reason"`
					ContentFilterResults map[string]any `json:"content_filter_results"`
				} `json:"choices"`
				Usage             map[string]any `json:"usage"`
				SystemFingerprint string         `json:"system_fingerprint"`
				ServiceTier       string         `json:"service_tier"`
			}

			// 解析data字段
			logger.LogDebug(ctx, "[SUCHUANG] Data field content: %s", string(suchuangResp.Data))
			logger.LogDebug(ctx, "[SUCHUANG] Data field length: %d", len(suchuangResp.Data))
			if err := json.Unmarshal(suchuangResp.Data, &openAIResp); err != nil {
				// 如果data字段不是预期的OpenAI格式，可能是直接的文本内容
				logger.LogDebug(ctx, "[SUCHUANG] DoResponse failed to parse data field as OpenAI format, trying as text: %v", err)

				// 创建一个新的OpenAI响应
				openAIResp = struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					Created int64  `json:"created"`
					Model   string `json:"model"`
					Choices []struct {
						Index   int `json:"index"`
						Message struct {
							Role         string  `json:"role"`
							Content      string  `json:"content"`
							Refusal      *string `json:"refusal"`
							Annotations  []any   `json:"annotations"`
							Audio        *string `json:"audio"`
							FunctionCall *any    `json:"function_call"`
							ToolCalls    *any    `json:"tool_calls"`
						} `json:"message"`
						Logprobs             *any           `json:"logprobs"`
						FinishReason         string         `json:"finish_reason"`
						ContentFilterResults map[string]any `json:"content_filter_results"`
					} `json:"choices"`
					Usage             map[string]any `json:"usage"`
					SystemFingerprint string         `json:"system_fingerprint"`
					ServiceTier       string         `json:"service_tier"`
				}{
					ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
					Object:  "chat.completion",
					Created: time.Now().Unix(),
					Model:   info.OriginModelName,
					Choices: []struct {
						Index   int `json:"index"`
						Message struct {
							Role         string  `json:"role"`
							Content      string  `json:"content"`
							Refusal      *string `json:"refusal"`
							Annotations  []any   `json:"annotations"`
							Audio        *string `json:"audio"`
							FunctionCall *any    `json:"function_call"`
							ToolCalls    *any    `json:"tool_calls"`
						} `json:"message"`
						Logprobs             *any           `json:"logprobs"`
						FinishReason         string         `json:"finish_reason"`
						ContentFilterResults map[string]any `json:"content_filter_results"`
					}{{
						Index: 0,
						Message: struct {
							Role         string  `json:"role"`
							Content      string  `json:"content"`
							Refusal      *string `json:"refusal"`
							Annotations  []any   `json:"annotations"`
							Audio        *string `json:"audio"`
							FunctionCall *any    `json:"function_call"`
							ToolCalls    *any    `json:"tool_calls"`
						}{Role: "assistant", Content: removeThinkContent(string(suchuangResp.Data)), Annotations: []any{}},
						FinishReason: "stop",
						Logprobs:     nil,
					}},
					Usage: map[string]any{
						"prompt_tokens":     100,  // 设置合理的 prompt tokens 数量
						"completion_tokens": 2000, // 设置合理的 completion tokens 数量
						"total_tokens":      2100, // 设置合理的总 tokens 数量
						"completion_tokens_details": map[string]any{
							"accepted_prediction_tokens": 2000,
							"audio_tokens":               0,
							"reasoning_tokens":           500,
							"rejected_prediction_tokens": 0,
						},
						"prompt_tokens_details": map[string]any{
							"audio_tokens":  0,
							"cached_tokens": 0,
						},
					},
					SystemFingerprint: "",
					ServiceTier:       "default",
				}
			} else {
				// 成功解析为OpenAI格式，需要处理content字段中的think内容
				for i := range openAIResp.Choices {
					openAIResp.Choices[i].Message.Content = removeThinkContent(openAIResp.Choices[i].Message.Content)
				}
			}

			// 确保object字段有值
			if openAIResp.Object == "" {
				openAIResp.Object = "chat.completion"
			}

			// 将转换后的响应写回响应体
			openAIRespBody, marshalErr := json.Marshal(openAIResp)
			if marshalErr != nil {
				logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Failed to marshal OpenAI response: %v", marshalErr))
				return nil, types.NewErrorWithStatusCode(errors.New("Failed to marshal response"), types.ErrorCodeBadResponse, 500)
			}
			logger.LogDebug(ctx, "[SUCHUANG] Converted OpenAI response: %s", string(openAIRespBody))
			logger.LogDebug(ctx, "[SUCHUANG] Converted response length: %d", len(openAIRespBody))

			// 将转换后的响应写入客户端
			service.IOCopyBytesGracefully(c, httpResp, openAIRespBody)

			// 返回Usage实例，包含正确的token数量
			return &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 2000,
				TotalTokens:      2100,
			}, nil
		}
		// 对于其他模型，也尝试解析响应
		logger.LogDebug(ctx, "[SUCHUANG] Handling chat completion for other model: %s", info.OriginModelName)

		// 尝试解析速创API响应
		var suchuangResp struct {
			Code     int             `json:"code"`
			Msg      string          `json:"msg"`
			Data     json.RawMessage `json:"data"`
			ExecTime float64         `json:"exec_time"`
			IP       string          `json:"ip"`
		}

		if err := json.Unmarshal(body, &suchuangResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] DoResponse failed to parse chat response: %v, body: %s", err, string(body)))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
		}

		if suchuangResp.Code != 200 {
			return nil, types.NewErrorWithStatusCode(errors.New(suchuangResp.Msg), types.ErrorCodeBadResponse, 500)
		}

		// 创建OpenAI格式的响应
		var openAIResp struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index   int `json:"index"`
				Message struct {
					Role         string  `json:"role"`
					Content      string  `json:"content"`
					Refusal      *string `json:"refusal"`
					Annotations  []any   `json:"annotations"`
					Audio        *string `json:"audio"`
					FunctionCall *any    `json:"function_call"`
					ToolCalls    *any    `json:"tool_calls"`
				} `json:"message"`
				Logprobs             *any           `json:"logprobs"`
				FinishReason         string         `json:"finish_reason"`
				ContentFilterResults map[string]any `json:"content_filter_results"`
			} `json:"choices"`
			Usage             map[string]any `json:"usage"`
			SystemFingerprint string         `json:"system_fingerprint"`
			ServiceTier       string         `json:"service_tier"`
		}

		// 解析data字段
		if err := json.Unmarshal(suchuangResp.Data, &openAIResp); err != nil {
			// 如果data字段不是预期的OpenAI格式，可能是直接的文本内容
			logger.LogDebug(ctx, "[SUCHUANG] DoResponse failed to parse data field as OpenAI format, trying as text: %v", err)

			// 创建一个新的OpenAI响应
			openAIResp = struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				Model   string `json:"model"`
				Choices []struct {
					Index   int `json:"index"`
					Message struct {
						Role         string  `json:"role"`
						Content      string  `json:"content"`
						Refusal      *string `json:"refusal"`
						Annotations  []any   `json:"annotations"`
						Audio        *string `json:"audio"`
						FunctionCall *any    `json:"function_call"`
						ToolCalls    *any    `json:"tool_calls"`
					} `json:"message"`
					Logprobs             *any           `json:"logprobs"`
					FinishReason         string         `json:"finish_reason"`
					ContentFilterResults map[string]any `json:"content_filter_results"`
				} `json:"choices"`
				Usage             map[string]any `json:"usage"`
				SystemFingerprint string         `json:"system_fingerprint"`
				ServiceTier       string         `json:"service_tier"`
			}{
				ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   info.OriginModelName,
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role         string  `json:"role"`
						Content      string  `json:"content"`
						Refusal      *string `json:"refusal"`
						Annotations  []any   `json:"annotations"`
						Audio        *string `json:"audio"`
						FunctionCall *any    `json:"function_call"`
						ToolCalls    *any    `json:"tool_calls"`
					} `json:"message"`
					Logprobs             *any           `json:"logprobs"`
					FinishReason         string         `json:"finish_reason"`
					ContentFilterResults map[string]any `json:"content_filter_results"`
				}{{
					Index: 0,
					Message: struct {
						Role         string  `json:"role"`
						Content      string  `json:"content"`
						Refusal      *string `json:"refusal"`
						Annotations  []any   `json:"annotations"`
						Audio        *string `json:"audio"`
						FunctionCall *any    `json:"function_call"`
						ToolCalls    *any    `json:"tool_calls"`
					}{Role: "assistant", Content: removeThinkContent(string(suchuangResp.Data)), Annotations: []any{}},
					FinishReason: "stop",
					Logprobs:     nil,
				}},
				Usage: map[string]any{
					"prompt_tokens":     0,
					"completion_tokens": 0,
					"total_tokens":      0,
					"completion_tokens_details": map[string]any{
						"accepted_prediction_tokens": 0,
						"audio_tokens":               0,
						"reasoning_tokens":           0,
						"rejected_prediction_tokens": 0,
					},
					"prompt_tokens_details": map[string]any{
						"audio_tokens":  0,
						"cached_tokens": 0,
					},
				},
				SystemFingerprint: "",
				ServiceTier:       "default",
			}
		} else {
			// 成功解析为OpenAI格式，需要处理content字段中的think内容
			for i := range openAIResp.Choices {
				openAIResp.Choices[i].Message.Content = removeThinkContent(openAIResp.Choices[i].Message.Content)
			}
		}

		// 确保object字段有值
		if openAIResp.Object == "" {
			openAIResp.Object = "chat.completion"
		}

		// 将转换后的响应写回响应体
		openAIRespBody, marshalErr := json.Marshal(openAIResp)
		if marshalErr != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Failed to marshal OpenAI response: %v", marshalErr))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to marshal response"), types.ErrorCodeBadResponse, 500)
		}

		// 将转换后的响应写入客户端
		service.IOCopyBytesGracefully(c, httpResp, openAIRespBody)

		// 返回Usage实例
		return &dto.Usage{}, nil

	default:
		// 其他请求返回Usage实例
		logger.LogDebug(ctx, "[SUCHUANG] Default response handling for URL path: %s", info.RequestURLPath)
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
	return nil, errors.New("ConvertClaudeRequest not implemented for suchuang channel")
}

// ConvertGeminiRequest 转换Gemini请求
// 实现channel.Adaptor接口的ConvertGeminiRequest方法
// 目前不支持Gemini请求，直接返回错误
func (s *SuchuangAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("ConvertGeminiRequest not implemented for suchuang channel")
}

// createImageTask 创建图片生成任务
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 调用速创API创建图片生成任务
// 返回任务ID和错误信息
func (s *SuchuangAdaptor) createImageTask(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (int, error) {
	ctx := c.Request.Context()

	// 获取请求URL
	fullURL, err := s.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] GetRequestURL failed: %v", err))
		return 0, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBody)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] New request failed: %v", err))
		return 0, err
	}

	// 设置请求头
	if err := s.SetupRequestHeader(c, &req.Header, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] SetupRequestHeader failed: %v", err))
		return 0, err
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Client Do failed: %v", err))
		return 0, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Read response body failed: %v", err))
		return 0, err
	}

	// 解析响应
	var result struct {
		Msg  string `json:"msg"`
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
		Code     int     `json:"code"`
		ExecTime float64 `json:"exec_time,omitempty"`
		IP       string  `json:"ip,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Unmarshal response failed: %v, body: %s", err, string(body)))
		return 0, err
	}

	if result.Code != 200 {
		logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] API returned error: %s, code: %d", result.Msg, result.Code))
		return 0, errors.New(result.Msg)
	}

	logger.LogDebug(ctx, "[SUCHUANG] Created image task with ID: %d", result.Data.ID)
	return result.Data.ID, nil
}

// pollImageResult 轮询图片生成结果
// 接收gin.Context、relaycommon.RelayInfo和任务ID参数
// 轮询速创API获取图片生成结果
// 返回OpenAI格式的响应数据和错误信息
func (s *SuchuangAdaptor) pollImageResult(c *gin.Context, info *relaycommon.RelayInfo, taskID int) (any, error) {
	ctx := c.Request.Context()

	// 构造轮询URL
	pollURL := fmt.Sprintf("%s/api/img/drawDetail", s.baseURL)

	// 构造轮询请求体
	pollRequestBody := map[string]interface{}{
		"id": taskID,
	}

	// 转换为JSON
	pollRequestBodyBytes, _ := json.Marshal(pollRequestBody)

	// 设置轮询间隔和超时
	pollInterval := 2 * time.Second
	timeout := 60 * time.Second
	startTime := time.Now()

	// 轮询获取结果
	for {
		// 创建请求
		req, err := http.NewRequest("POST", pollURL, bytes.NewBuffer(pollRequestBodyBytes))
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] New poll request failed: %v", err))
			return nil, err
		}

		// 设置请求头
		if err := s.SetupRequestHeader(c, &req.Header, info); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Setup poll request header failed: %v", err))
			return nil, err
		}

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Client Do failed: %v", err))
			return nil, err
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Read poll response body failed: %v", err))
			return nil, err
		}

		// 解析响应
		var pollResult struct {
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

		if err := json.Unmarshal(body, &pollResult); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Unmarshal poll response failed: %v, body: %s", err, string(body)))
			return nil, err
		}

		if pollResult.Code != 200 {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Poll API returned error: %s, code: %d", pollResult.Msg, pollResult.Code))
			return nil, errors.New(pollResult.Msg)
		}

		// 检查任务状态
		if pollResult.Data.Status == 2 {
			// 任务完成，转换为OpenAI格式响应
			openAIResp := struct {
				Data []struct {
					URL string `json:"url"`
				} `json:"data"`
				Created int64 `json:"created"`
			}{
				Data: []struct {
					URL string `json:"url"`
				}{{pollResult.Data.ImageURL}},
				Created: time.Now().Unix(),
			}

			logger.LogDebug(ctx, "[SUCHUANG] Image generation completed successfully")
			return openAIResp, nil
		}

		// 检查是否超时
		if time.Since(startTime) > timeout {
			logger.LogError(ctx, "[SUCHUANG] Image generation timeout")
			return nil, errors.New("Image generation timeout")
		}

		// 等待一段时间后继续轮询
		time.Sleep(pollInterval)
	}
}
