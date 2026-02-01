package yushu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunshuAPI/common"
	"yunshuAPI/dto"
	"yunshuAPI/logger"
	"yunshuAPI/model"
	relaycommon "yunshuAPI/relay/common"
	"yunshuAPI/service"
	"yunshuAPI/types"

	"github.com/gin-gonic/gin"
)

// ChannelName 渠道名称
const ChannelName = "yushu"

// ModelList 模型列表
var ModelList = []string{
	"sora-2",
	"sora-2-all",
}

// YushuAdaptor 云舒渠道适配器
// 用于处理云舒API的请求和响应适配
// 实现了channel.Adaptor接口
// 支持视频生成功能
// 创建任务接口：/v1/video/create
// 轮询结果接口：/v1/video/query
// 响应需要转换为OpenAI格式
// 所有请求头中的Authorization不需要Bearer拼接
type YushuAdaptor struct {
	channelType int
	baseURL     string
	apiKey      string
}

// Adaptor 实现channel.Adaptor接口的适配器类
type Adaptor struct {
	YushuAdaptor
}

// TaskAdaptor 实现channel.TaskAdaptor接口的适配器类
type TaskAdaptor struct {
	YushuAdaptor
}

// NewYushuAdaptor 创建云舒渠道适配器实例
// 接收渠道类型、基础URL和API密钥作为参数
// 返回YushuAdaptor实例
func NewYushuAdaptor(channelType int, baseURL, apiKey string) *YushuAdaptor {
	return &YushuAdaptor{
		channelType: channelType,
		baseURL:     baseURL,
		apiKey:      apiKey,
	}
}

// Init 初始化适配器
// 实现channel.Adaptor接口的Init方法
// 接收relaycommon.RelayInfo参数
// 从参数中获取渠道类型、基础URL和API密钥
func (y *YushuAdaptor) Init(info *relaycommon.RelayInfo) {
	y.channelType = info.ChannelType
	y.baseURL = info.ChannelBaseUrl
	y.apiKey = info.ApiKey
}

// GetRequestURL 获取请求URL
// 实现channel.Adaptor接口的GetRequestURL方法
// 接收relaycommon.RelayInfo参数
// 根据请求路径和渠道类型构建完整URL
func (y *YushuAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 检查DEBUG模式是否启用
	if common.DebugEnabled {
		fmt.Printf("[YUSHU] GetRequestURL called with: baseURL=%s, RequestURLPath=%s, Model=%s\n", y.baseURL, info.RequestURLPath, info.OriginModelName)
	}

	// 如果是视频生成请求，返回视频生成API端点
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		url := fmt.Sprintf("%s/v1/video/create", y.baseURL)
		if common.DebugEnabled {
			fmt.Printf("[YUSHU] Generated video generation URL: %s\n", url)
		}
		return url, nil
	}

	// 如果是轮询请求，返回轮询API端点
	if strings.HasPrefix(info.RequestURLPath, "/v1/video/query") {
		url := fmt.Sprintf("%s/v1/video/query", y.baseURL)
		if common.DebugEnabled {
			fmt.Printf("[YUSHU] Generated poll URL: %s\n", url)
		}
		return url, nil
	}

	// 其他请求路径返回错误
	if common.DebugEnabled {
		fmt.Printf("[YUSHU] Unsupported request URL path: %s\n", info.RequestURLPath)
	}
	return "", errors.New("unsupported request URL path for yushu channel")
}

// SetupRequestHeader 设置请求头
// 实现channel.Adaptor接口的SetupRequestHeader方法
// 接收gin.Context、http.Header和relaycommon.RelayInfo参数
// 设置Content-Type和Authorization头
// Authorization头需要Bearer拼接
func (y *YushuAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	// 设置Content-Type为application/json
	req.Set("Content-Type", "application/json")
	// 设置Authorization头，需要Bearer拼接
	req.Set("Authorization", "Bearer "+y.apiKey)
	return nil
}

// ConvertOpenAIRequest 转换OpenAI请求到云舒API请求
// 实现channel.Adaptor接口的ConvertOpenAIRequest方法
// 接收gin.Context、relaycommon.RelayInfo、dto.GeneralOpenAIRequest参数
// 支持视频生成请求转换
func (y *YushuAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	ctx := c.Request.Context()

	// 支持视频生成请求
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		logger.LogDebug(ctx, "[YUSHU] Converting video generation request for %s", request.Model)

		// 定义视频请求结构体
		var videoRequest struct {
			Model          string   `json:"model"`
			Prompt         string   `json:"prompt"`
			Seconds        string   `json:"seconds"`
			Size           string   `json:"size"`
			InputReference []string `json:"input_reference"`
		}

		// 将GeneralOpenAIRequest转换为map以便获取所有字段
		requestMap := request.ToMap()

		// 从requestMap中提取所有字段
		if model, ok := requestMap["model"].(string); ok {
			videoRequest.Model = model
		} else {
			videoRequest.Model = request.Model
		}

		if prompt, ok := requestMap["prompt"].(string); ok {
			videoRequest.Prompt = prompt
		} else if request.Prompt != nil {
			if promptStr, ok := request.Prompt.(string); ok {
				videoRequest.Prompt = promptStr
			}
		}

		if seconds, ok := requestMap["seconds"].(string); ok {
			videoRequest.Seconds = seconds
		} else if secondsFloat, ok := requestMap["seconds"].(float64); ok {
			videoRequest.Seconds = fmt.Sprintf("%.0f", secondsFloat)
		} else if secondsInt, ok := requestMap["seconds"].(int); ok {
			videoRequest.Seconds = fmt.Sprintf("%d", secondsInt)
		}

		if size, ok := requestMap["size"].(string); ok {
			videoRequest.Size = size
		}

		if inputReference, ok := requestMap["input_reference"].([]string); ok {
			videoRequest.InputReference = inputReference
		} else if inputReferenceInterface, ok := requestMap["input_reference"].([]any); ok {
			for _, ref := range inputReferenceInterface {
				if refStr, ok := ref.(string); ok {
					videoRequest.InputReference = append(videoRequest.InputReference, refStr)
				}
			}
		}

		// 设置默认�?
		if videoRequest.Model == "" {
			videoRequest.Model = request.Model
		}

		// 构建云舒API请求�?
		return y.buildYushuVideoRequest(videoRequest)
	}

	return nil, errors.New("unsupported request URL path for yushu channel")
}

// buildYushuVideoRequest 构建云舒视频生成请求
func (y *YushuAdaptor) buildYushuVideoRequest(videoRequest struct {
	Model          string   `json:"model"`
	Prompt         string   `json:"prompt"`
	Seconds        string   `json:"seconds"`
	Size           string   `json:"size"`
	InputReference []string `json:"input_reference"`
}) (map[string]interface{}, error) {
	// 转换seconds到duration
	duration := 15 // 默认15�?
	if secondsStr := videoRequest.Seconds; secondsStr != "" {
		if d, err := strconv.Atoi(secondsStr); err == nil {
			duration = d
		}
	}

	// 转换size到orientation
	orientation := "landscape" // 默认横屏
	sizeStr := videoRequest.Size
	if sizeStr == "720x1280" || sizeStr == "720×1280" || sizeStr == "1024x1792" || sizeStr == "1024×1792" {
		orientation = "portrait"
	}

	// 处理input_reference到images
	images := []string{}
	for _, ref := range videoRequest.InputReference {
		// 清理URL中的反引号和空格
		cleanedRef := strings.Trim(ref, " `")
		if cleanedRef != "" {
			images = append(images, cleanedRef)
		}
	}

	// 构建云舒API请求�?
	yushuRequest := map[string]interface{}{
		"images":      images,
		"model":       "sora-2-all", // 固定模型名称
		"orientation": orientation,
		"prompt":      videoRequest.Prompt,
		"size":        "large", // 固定为large
		"duration":    duration,
		"watermark":   true, // 固定为true
	}

	return yushuRequest, nil
}

// ConvertRerankRequest 转换重排请求
// 实现channel.Adaptor接口的ConvertRerankRequest方法
// 目前不支持重排请求，直接返回错误
func (y *YushuAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("Rerank not implemented for yushu channel")
}

// ConvertEmbeddingRequest 转换嵌入请求
// 实现channel.Adaptor接口的ConvertEmbeddingRequest方法
// 目前不支持嵌入请求，直接返回错误
func (y *YushuAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("Embedding not implemented for yushu channel")
}

// ConvertAudioRequest 转换音频请求
// 实现channel.Adaptor接口的ConvertAudioRequest方法
// 目前不支持音频请求，直接返回错误
func (y *YushuAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("Audio not implemented for yushu channel")
}

// ConvertImageRequest 转换图片生成请求
// 实现channel.Adaptor接口的ConvertImageRequest方法
// 目前不支持图片生成请求，直接返回错误
func (y *YushuAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("Image generation not implemented for yushu channel")
}

// ConvertOpenAIResponsesRequest 转换OpenAI响应请求
// 实现channel.Adaptor接口的ConvertOpenAIResponsesRequest方法
// 目前不支持此功能，直接返回错�?
func (y *YushuAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("OpenAIResponses not implemented for yushu channel")
}

// DoRequest 执行请求
// 实现channel.TaskAdaptor接口的DoRequest方法
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 执行HTTP请求并返回响�?
func (y *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[YUSHU] DoRequest called with RequestURLPath: %s", info.RequestURLPath)

	// 读取并记录请求体
	var requestBodyCopy io.Reader
	var bodyBytes []byte
	if requestBody != nil {
		bodyBytes, _ = io.ReadAll(requestBody)
		logger.LogDebug(ctx, "[YUSHU] Original request body: %s", string(bodyBytes))
		requestBodyCopy = io.NopCloser(bytes.NewBuffer(bodyBytes))
	} else {
		requestBodyCopy = nil
	}

	// 构建请求URL
	fullURL, err := y.BuildRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] BuildRequestURL failed: %v", err))
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBodyCopy)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] New request failed: %v", err))
		return nil, err
	}

	// 设置请求�?
	if err := y.BuildRequestHeader(c, req, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] BuildRequestHeader failed: %v", err))
		return nil, err
	}

	// 发送请�?
	client := service.GetHttpClient()
	logger.LogDebug(ctx, "[YUSHU] Sending request to URL: %s with method: %s", req.URL.String(), req.Method)
	logger.LogDebug(ctx, "[YUSHU] Request headers: %v", req.Header)
	logger.LogDebug(ctx, "[YUSHU] Request body: %s", string(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] Client Do failed: %v", err))
		return nil, err
	}

	// 读取响应体以便记录日�?
	respBody, _ := io.ReadAll(resp.Body)
	logger.LogDebug(ctx, "[YUSHU] Response status: %d", resp.StatusCode)
	logger.LogDebug(ctx, "[YUSHU] Response headers: %v", resp.Header)
	logger.LogDebug(ctx, "[YUSHU] Response body: %s", string(respBody))

	// 重置响应体，以便后续处理
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	logger.LogDebug(ctx, "[YUSHU] Request completed successfully, returning *http.Response")
	return resp, nil
}

// DoRequest 执行请求
// 实现channel.Adaptor接口的DoRequest方法
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 执行HTTP请求并返回响�?
func (y *YushuAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[YUSHU] DoRequest called with RequestURLPath: %s", info.RequestURLPath)

	// 读取并记录请求体
	var requestBodyCopy io.Reader
	var bodyBytes []byte
	if requestBody != nil {
		bodyBytes, _ = io.ReadAll(requestBody)
		logger.LogDebug(ctx, "[YUSHU] Original request body: %s", string(bodyBytes))
		requestBodyCopy = io.NopCloser(bytes.NewBuffer(bodyBytes))
	} else {
		requestBodyCopy = nil
	}

	// 所有请求统一使用GetRequestURL获取正确的URL
	fullURL, err := y.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] GetRequestURL failed: %v", err))
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBodyCopy)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] New request failed: %v", err))
		return nil, err
	}

	// 设置请求�?
	if err := y.SetupRequestHeader(c, &req.Header, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] SetupRequestHeader failed: %v", err))
		return nil, err
	}

	// 发送请�?
	client := service.GetHttpClient()
	logger.LogDebug(ctx, "[YUSHU] Sending request to URL: %s with method: %s", req.URL.String(), req.Method)
	logger.LogDebug(ctx, "[YUSHU] Request headers: %v", req.Header)
	logger.LogDebug(ctx, "[YUSHU] Request body: %s", string(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[YUSHU] Client Do failed: %v", err))
		return nil, err
	}

	// 读取响应体以便记录日�?
	respBody, _ := io.ReadAll(resp.Body)
	logger.LogDebug(ctx, "[YUSHU] Response status: %d", resp.StatusCode)
	logger.LogDebug(ctx, "[YUSHU] Response headers: %v", resp.Header)
	logger.LogDebug(ctx, "[YUSHU] Response body: %s", string(respBody))

	// 重置响应体，以便DoResponse方法使用
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	logger.LogDebug(ctx, "[YUSHU] Request completed successfully, returning *http.Response")
	return resp, nil
}

// DoResponse 处理响应
// 实现channel.Adaptor接口的DoResponse方法
// 接收gin.Context、http.Response和relaycommon.RelayInfo参数
// 处理响应并转换为OpenAI格式
func (y *YushuAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	ctx := c.Request.Context()

	// 处理响应
	var body []byte
	var httpResp = resp

	// 如果响应为空，直接返回空的usage信息和nil错误
	if resp == nil {
		logger.LogDebug(ctx, "[YUSHU] Response is nil, returning empty usage")
		return &dto.Usage{}, nil
	}

	// 读取响应�?
	var readErr error
	body, readErr = io.ReadAll(httpResp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(errors.New("Failed to read response body"), types.ErrorCodeBadResponse, 500)
	}

	// 如果响应体为空，直接返回空的usage信息和nil错误
	if len(body) == 0 {
		logger.LogDebug(ctx, "[YUSHU] Response body is empty, returning empty usage")
		return &dto.Usage{}, nil
	}

	logger.LogDebug(ctx, "[YUSHU] Response body: %s", string(body))

	// 重置响应体，以便后续处理
	if httpResp != nil {
		httpResp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// 处理视频生成响应
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		// 解析云舒API响应
		var yushuResp struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			StatusUpdateTime int64  `json:"status_update_time"`
			Error            struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Size             string `json:"size"`
			Model            string `json:"model"`
			Object           string `json:"object"`
			Seconds          string `json:"seconds"`
			Progress         int    `json:"progress"`
			VideoURL         string `json:"video_url"`
			CreatedAt        int64  `json:"created_at"`
			CompletedAt      int64  `json:"completed_at"`
		}

		if err := json.Unmarshal(body, &yushuResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[YUSHU] DoResponse failed to parse video response: %v, body: %s", err, string(body)))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
		}

		// 构造OpenAI格式的视频生成响�?
		videoResponse := &dto.OpenAIVideo{
			ID:        yushuResp.ID,
			Status:    yushuResp.Status,
			CreatedAt: time.Now().UnixNano() / int64(time.Millisecond), // 使用当前时间
			Model:     "sora-2",
			Object:    "video",
			Seconds:   yushuResp.Seconds,
			Progress:  yushuResp.Progress,
		}

		// 如果视频生成完成，添加视频URL到Metadata
		if yushuResp.Status == "completed" && yushuResp.VideoURL != "" {
			videoResponse.SetMetadata("video_url", strings.Trim(yushuResp.VideoURL, " `"))
		}

		// 将转换后的响应写回响应体
		if httpResp != nil {
			videoRespBody, _ := json.Marshal(videoResponse)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(videoRespBody))
		}

		return &dto.Usage{}, nil
	}

	// 处理轮询结果响应
	if strings.HasPrefix(info.RequestURLPath, "/v1/video/query") {
		// 解析云舒API轮询响应
		var pollResp struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			StatusUpdateTime int64  `json:"status_update_time"`
			Error            struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Size             string `json:"size"`
			Model            string `json:"model"`
			Object           string `json:"object"`
			Seconds          string `json:"seconds"`
			Progress         int    `json:"progress"`
			VideoURL         string `json:"video_url"`
			CreatedAt        int64  `json:"created_at"`
			CompletedAt      int64  `json:"completed_at"`
		}

		if err := json.Unmarshal(body, &pollResp); err != nil {
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse poll response"), types.ErrorCodeBadResponse, 500)
		}

		// 构造OpenAI格式的轮询响�?
		videoResponse := &dto.OpenAIVideo{
			ID:        pollResp.ID,
			Status:    pollResp.Status,
			CreatedAt: time.Now().UnixNano() / int64(time.Millisecond),
			Model:     "sora-2",
			Object:    "video",
			Seconds:   pollResp.Seconds,
			Size:      pollResp.Size,
		}

		// 设置进度
		switch pollResp.Status {
		case "failed":
			videoResponse.Progress = 0
		case "pending":
			videoResponse.Progress = pollResp.Progress
		case "queued":
			videoResponse.Progress = 0
		case "completed":
			videoResponse.Progress = 100
		default:
			videoResponse.Progress = pollResp.Progress
		}

		// 如果有错误信息，添加错误
		if pollResp.Error.Message != "" {
			videoResponse.Error = &dto.OpenAIVideoError{
				Code:    pollResp.Error.Code,
				Message: pollResp.Error.Message,
			}
		}

		// 如果视频生成完成，添加视频URL到Metadata
		if pollResp.Status == "completed" && pollResp.VideoURL != "" {
			videoResponse.SetMetadata("video_url", strings.Trim(pollResp.VideoURL, " `"))
		}

		// 将转换后的响应写回响应体
		if httpResp != nil {
			openAIRespBody, _ := json.Marshal(videoResponse)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))
		}

		return &dto.Usage{}, nil
	}

	// 其他请求返回空的usage信息
	logger.LogDebug(ctx, "[YUSHU] Default response handling for URL path: %s", info.RequestURLPath)
	return &dto.Usage{}, nil
}

// GetModelList 获取支持的模型列�?
// 实现channel.Adaptor接口的GetModelList方法
// 返回ModelList变量
func (y *YushuAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 获取渠道名称
// 实现channel.Adaptor接口的GetChannelName方法
// 返回ChannelName常量
func (y *YushuAdaptor) GetChannelName() string {
	return ChannelName
}

// ConvertClaudeRequest 转换Claude请求
// 实现channel.Adaptor接口的ConvertClaudeRequest方法
// 目前不支持Claude请求，直接返回错�?
func (y *YushuAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("ConvertClaudeRequest not implemented for yushu channel")
}

// ConvertGeminiRequest 转换Gemini请求
// 实现channel.Adaptor接口的ConvertGeminiRequest方法
// 目前不支持Gemini请求，直接返回错�?
func (y *YushuAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("ConvertGeminiRequest not implemented for yushu channel")
}

// ============================// TaskAdaptor Implementation// ============================

// ValidateRequestAndSetAction 验证请求并设置操�?
// 实现channel.TaskAdaptor接口的ValidateRequestAndSetAction方法
func (t *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	// 验证请求并设置操�?
	return relaycommon.ValidateMultipartDirect(c, info)
}

// BuildRequestURL 构建请求URL
// 实现channel.TaskAdaptor接口的BuildRequestURL方法
func (t *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 构建视频生成API URL
	url := fmt.Sprintf("%s/v1/video/create", t.baseURL)
	return url, nil
}

// BuildRequestHeader 构建请求�?
// 实现channel.TaskAdaptor接口的BuildRequestHeader方法
func (t *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	// 设置请求头，需要Bearer拼接
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// BuildRequestBody 构建请求�?
// 实现channel.TaskAdaptor接口的BuildRequestBody方法
func (t *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	// 获取任务请求
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	// 构建视频请求结构�?
	var videoRequest struct {
		Model          string   `json:"model"`
		Prompt         string   `json:"prompt"`
		Seconds        string   `json:"seconds"`
		Size           string   `json:"size"`
		InputReference []string `json:"input_reference"`
	}

	// 从TaskSubmitReq中提取字�?
	videoRequest.Model = req.Model
	videoRequest.Prompt = req.Prompt
	videoRequest.Seconds = req.Seconds
	videoRequest.Size = req.Size

	// 处理InputReference字段
	if req.InputReference != nil {
		switch v := req.InputReference.(type) {
		case string:
			videoRequest.InputReference = []string{v}
		case []string:
			videoRequest.InputReference = v
		case []interface{}:
			for _, ref := range v {
				if refStr, ok := ref.(string); ok {
					videoRequest.InputReference = append(videoRequest.InputReference, refStr)
				}
			}
		}
	}

	// 构建云舒API请求�?
	yushuRequest, err := t.buildYushuVideoRequest(videoRequest)
	if err != nil {
		return nil, err
	}

	// 转换为JSON
	requestBody, err := json.Marshal(yushuRequest)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(requestBody), nil
}

// DoResponse 处理响应
// 实现channel.TaskAdaptor接口的DoResponse方法
func (t *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	// 处理响应
	var body []byte
	var readErr error
	body, readErr = io.ReadAll(resp.Body)
	if readErr != nil {
		return "", nil, service.TaskErrorWrapper(readErr, "failed_to_read_response", 500)
	}

	// 解析云舒API响应
	var yushuResp struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Progress  int    `json:"progress"`
		CreatedAt int64  `json:"created_at"`
		Seconds   string `json:"seconds"`
		Size      string `json:"size"`
	}

	if err := json.Unmarshal(body, &yushuResp); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "failed_to_parse_response", 500)
	}

	// 返回任务ID和响应体
	return yushuResp.ID, body, nil
}

// ProcessResponse 处理响应
// 实现channel.TaskAdaptor接口的ProcessResponse方法
func (t *TaskAdaptor) ProcessResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, *dto.TaskError) {
	// 处理响应
	var body []byte
	var readErr error
	body, readErr = io.ReadAll(resp.Body)
	if readErr != nil {
		return "", service.TaskErrorWrapper(readErr, "failed_to_read_response", 500)
	}

	// 解析云舒API响应
	var yushuResp struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Progress  int    `json:"progress"`
		CreatedAt int64  `json:"created_at"`
		Seconds   string `json:"seconds"`
		Size      string `json:"size"`
	}

	if err := json.Unmarshal(body, &yushuResp); err != nil {
		return "", service.TaskErrorWrapper(err, "failed_to_parse_response", 500)
	}

	// 返回任务ID
	return yushuResp.ID, nil
}

// ParseTaskResult 解析任务结果
// 实现channel.TaskAdaptor接口的ParseTaskResult方法
func (t *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// 解析轮询响应
	var pollResp struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		Progress         int    `json:"progress"`
		VideoURL         string `json:"video_url"`
		CompletedAt      int64  `json:"completed_at"`
		StatusUpdateTime int64  `json:"status_update_time"`
		Error            struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &pollResp); err != nil {
		return nil, err
	}

	// 转换状态和进度
	status := pollResp.Status
	progress := ""
	
	// 根据状态设置进�?
	switch status {
	case "failed":
		progress = "0%"
	case "pending":
		progress = fmt.Sprintf("%d%%", pollResp.Progress)
	case "queued":
		progress = "0%"
	case "completed":
		progress = "100%"
	}

	// 构建任务信息
	taskInfo := &relaycommon.TaskInfo{
		TaskID:   pollResp.ID,
		Status:   status,
		Progress: progress,
		Url:      strings.Trim(pollResp.VideoURL, " `"),
	}

	return taskInfo, nil
}

// BuildTaskInfo 构建任务信息
// 实现channel.TaskAdaptor接口的BuildTaskInfo方法
func (t *TaskAdaptor) BuildTaskInfo(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*relaycommon.TaskInfo, error) {
	// 读取响应�?
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var yushuResp struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Progress  int    `json:"progress"`
		CreatedAt int64  `json:"created_at"`
		Seconds   string `json:"seconds"`
		Size      string `json:"size"`
	}

	if err := json.Unmarshal(body, &yushuResp); err != nil {
		return nil, err
	}

	// 构建任务信息
	return &relaycommon.TaskInfo{
		TaskID:   yushuResp.ID,
		Status:   yushuResp.Status,
		Progress: fmt.Sprintf("%d%%", yushuResp.Progress),
	},
		nil
}

// ConvertToOpenAIVideo 转换为OpenAI视频格式
// 实现channel.OpenAIVideoConverter接口的ConvertToOpenAIVideo方法
func (t *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	// 解析任务数据
	var taskData struct {
		Status    string `json:"status"`
		Progress  int    `json:"progress"`
		VideoURL  string `json:"video_url"`
		Error     struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		// 如果解析失败，使用任务的基本信息
		taskData.Status = string(task.Status)
		taskData.Progress, _ = strconv.Atoi(strings.TrimSuffix(task.Progress, "%"))
	}

	// 构建OpenAI视频响应
	openAIVideo := &dto.OpenAIVideo{
		ID:        task.TaskID,
		Object:    "video",
		Model:     "sora-2",
		Status:    taskData.Status,
		Progress:  taskData.Progress,
		CreatedAt: task.SubmitTime * 1000, // 转换为毫�?
	}

	// 如果有错误信息，添加错误
	if taskData.Error.Message != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Code:    taskData.Error.Code,
			Message: taskData.Error.Message,
		}
	}

	// 将响应转换为JSON
	respBody, err := json.Marshal(openAIVideo)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// FetchTask 获取任务状�?
// 实现channel.TaskAdaptor接口的FetchTask方法
func (t *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskId, ok := body["task_id"].(string)
	if !ok {
		return nil, errors.New("task_id not found in body")
	}

	// 构建轮询URL
	pollURL := fmt.Sprintf("%s/v1/video/query?id=%s", baseUrl, taskId)

	// 创建请求
	req, err := http.NewRequest("GET", pollURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求�?
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	// 发送请�?
	client := service.GetHttpClient()
	return client.Do(req)
}
