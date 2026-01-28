package kieai

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

	"xunkecloudAPI/common"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/logger"
	"xunkecloudAPI/relay/channel"
	relaycommon "xunkecloudAPI/relay/common"
	"xunkecloudAPI/service"
	"xunkecloudAPI/types"

	"github.com/gin-gonic/gin"
)

// ChannelName 渠道名称
const ChannelName = "kieai"

// ModelList 模型列表
var ModelList = []string{
	"sora-2",
}

// KieaiAdaptor kieAi渠道适配器
// 用于处理kieAi API的请求和响应适配
// 实现了channel.Adaptor接口
// 支持视频生成功能
// 创建任务接口：POST https://api.kie.ai/api/v1/jobs/createTask
// 轮询结果接口：GET https://api.kie.ai/api/v1/jobs/recordInfo?taskId=
// 响应需要转换为OpenAI格式
// 请求头中的Authorization需要Bearer拼接

type KieaiAdaptor struct {
	channelType int
	baseURL     string
	apiKey      string
}

// Adaptor 实现channel.Adaptor接口的适配器类型
type Adaptor struct {
	KieaiAdaptor
}

// TaskAdaptor 实现channel.TaskAdaptor接口的适配器类型
type TaskAdaptor struct {
	KieaiAdaptor
}

// NewKieaiAdaptor 创建kieAi渠道适配器实例
// 接收渠道类型、基础URL和API密钥作为参数
// 返回KieaiAdaptor实例
func NewKieaiAdaptor(channelType int, baseURL, apiKey string) *KieaiAdaptor {
	return &KieaiAdaptor{
		channelType: channelType,
		baseURL:     baseURL,
		apiKey:      apiKey,
	}
}

// Init 初始化适配器
// 实现channel.Adaptor接口的Init方法
// 接收relaycommon.RelayInfo参数
// 从参数中获取渠道类型、基础URL和API密钥
func (k *KieaiAdaptor) Init(info *relaycommon.RelayInfo) {
	k.channelType = info.ChannelType
	k.baseURL = info.ChannelBaseUrl
	k.apiKey = info.ApiKey
}

// GetRequestURL 获取请求URL
// 实现channel.Adaptor接口的GetRequestURL方法
// 接收relaycommon.RelayInfo参数
// 根据请求路径构建完整URL
func (k *KieaiAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 检查DEBUG模式是否启用
	if common.DebugEnabled {
		// 直接输出到控制台以确保我们能看到日志
		fmt.Println("[KIEAI] DEBUG MODE IS ENABLED")
		modelName := info.OriginModelName
		upstreamModelName := ""
		if info.ChannelMeta != nil && info.ChannelMeta.UpstreamModelName != "" {
			upstreamModelName = info.ChannelMeta.UpstreamModelName
		} else if info.UpstreamModelName != "" {
			upstreamModelName = info.UpstreamModelName
		}
		if upstreamModelName != "" {
			fmt.Printf("[KIEAI] GetRequestURL called with: baseURL=%s, RequestURLPath=%s, Model=%s, UpstreamModel=%s\n", k.baseURL, info.RequestURLPath, modelName, upstreamModelName)
		} else {
			fmt.Printf("[KIEAI] GetRequestURL called with: baseURL=%s, RequestURLPath=%s, Model=%s\n", k.baseURL, info.RequestURLPath, modelName)
		}
	}

	// 根据请求路径返回不同的API端点
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		// 视频生成请求
		url := "https://api.kie.ai/api/v1/jobs/createTask"
		if common.DebugEnabled {
			fmt.Printf("[KIEAI] Generated video generation URL: %s\n", url)
		}
		return url, nil
	} else if strings.HasPrefix(info.RequestURLPath, "/api/jobs/recordInfo") {
		// 轮询请求
		url := info.RequestURLPath
		if common.DebugEnabled {
			fmt.Printf("[KIEAI] Generated poll URL: %s\n", url)
		}
		return url, nil
	}

	// 其他请求路径返回错误
	if common.DebugEnabled {
		fmt.Printf("[KIEAI] Unsupported request URL path: %s\n", info.RequestURLPath)
	}
	return "", errors.New("unsupported request URL path for kieai channel")
}

// SetupRequestHeader 设置请求头
// 实现channel.Adaptor接口的SetupRequestHeader方法
// 接收gin.Context、http.Header和relaycommon.RelayInfo参数
// 设置Content-Type和Authorization头
// Authorization头需要Bearer拼接
func (k *KieaiAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	// 设置Content-Type为application/json
	req.Set("Content-Type", "application/json")
	// 设置Authorization头，需要Bearer拼接
	req.Set("Authorization", "Bearer "+k.apiKey)
	return nil
}

// ConvertOpenAIRequest 转换OpenAI请求到kieAi API请求
// 实现channel.Adaptor接口的ConvertOpenAIRequest方法
// 接收gin.Context、relaycommon.RelayInfo和*dto.GeneralOpenAIRequest参数
// 支持视频生成请求转换
func (k *KieaiAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	ctx := c.Request.Context()

	// 支持sora-2视频生成请求
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		logger.LogDebug(ctx, "[KIEAI] Converting video generation request for %s", request.Model)

		// 定义视频请求结构体
		var videoRequest struct {
			Model          string   `json:"model"`
			Prompt         string   `json:"prompt"`
			Seconds        string   `json:"seconds"`
			Size           string   `json:"size"`
			InputReference []string `json:"input_reference"`
		}

		// 直接从gin上下文获取原始请求体
		var rawRequest map[string]any
		if err := c.ShouldBindJSON(&rawRequest); err == nil {
			// 从原始请求体中提取所有字段
			if model, ok := rawRequest["model"].(string); ok {
				videoRequest.Model = model
			} else {
				// 从request对象获取model
				videoRequest.Model = request.Model
			}

			if prompt, ok := rawRequest["prompt"].(string); ok {
				videoRequest.Prompt = prompt
			} else if request.Prompt != nil {
				// 从request对象获取prompt
				if promptStr, ok := request.Prompt.(string); ok {
					videoRequest.Prompt = promptStr
				}
			}

			// 处理Seconds字段，支持多种类型
			if seconds, ok := rawRequest["seconds"].(string); ok {
				videoRequest.Seconds = seconds
			} else if secondsFloat, ok := rawRequest["seconds"].(float64); ok {
				videoRequest.Seconds = fmt.Sprintf("%.0f", secondsFloat)
			} else if secondsInt, ok := rawRequest["seconds"].(int); ok {
				videoRequest.Seconds = fmt.Sprintf("%d", secondsInt)
			} else {
				// 默认值
				videoRequest.Seconds = "15"
			}

			if size, ok := rawRequest["size"].(string); ok {
				videoRequest.Size = size
			} else {
				// 默认值
				videoRequest.Size = "720x1280"
			}

			// 处理InputReference字段
			if inputReference, ok := rawRequest["input_reference"].([]string); ok {
				videoRequest.InputReference = inputReference
			} else if inputReferenceInterface, ok := rawRequest["input_reference"].([]any); ok {
				for _, ref := range inputReferenceInterface {
					if refStr, ok := ref.(string); ok {
						videoRequest.InputReference = append(videoRequest.InputReference, refStr)
					}
				}
			}
		} else {
			// 如果无法绑定原始请求体，使用request对象的字段
			videoRequest.Model = request.Model
			if request.Prompt != nil {
				if promptStr, ok := request.Prompt.(string); ok {
					videoRequest.Prompt = promptStr
				}
			}
			videoRequest.Size = "720x1280"
			videoRequest.Seconds = "15"
		}

		// 构建kieAi API请求体
		return k.buildKieaiVideoRequest(videoRequest)
	}

	// 其他请求返回错误
	return nil, errors.New("unsupported request type for kieai channel")
}

// buildKieaiVideoRequest 构建kieAi视频生成请求
func (k *KieaiAdaptor) buildKieaiVideoRequest(videoRequest struct {
	Model          string   `json:"model"`
	Prompt         string   `json:"prompt"`
	Seconds        string   `json:"seconds"`
	Size           string   `json:"size"`
	InputReference []string `json:"input_reference"`
}) (map[string]interface{}, error) {
	// 确定模型类型
	modelType := "sora-2-text-to-video"
	if len(videoRequest.InputReference) > 0 {
		modelType = "sora-2-image-to-video"
	}

	// 确定aspect_ratio
	aspectRatio := "landscape"
	size := videoRequest.Size
	if size == "720x1280" || size == "720×1280" || size == "1024x1792" || size == "1024×1792" {
		aspectRatio = "portrait"
	}

	// 处理image_urls
	imageUrls := []string{}
	for _, ref := range videoRequest.InputReference {
		// 清理URL中的反引号和空格
		cleanUrl := strings.Trim(ref, " `")
		if cleanUrl != "" {
			imageUrls = append(imageUrls, cleanUrl)
		}
	}

	// 处理n_frames - 直接使用传入的seconds值（只允许"10"或"15"）
	nFrames := "15" // 默认值
	if secondsStr := videoRequest.Seconds; secondsStr != "" {
		// 检查值是否为允许的选项
		if secondsStr == "10" || secondsStr == "15" {
			nFrames = secondsStr
		}
	}

	// 构建请求体
	kieaiRequest := map[string]interface{}{
		"model": modelType,
		"input": map[string]interface{}{
			"prompt":           videoRequest.Prompt,
			"image_urls":       imageUrls,
			"aspect_ratio":     aspectRatio,
			"n_frames":         nFrames,
			"remove_watermark": true,
		},
	}

	return kieaiRequest, nil
}

// ConvertRerankRequest 转换重排请求
// 实现channel.Adaptor接口的ConvertRerankRequest方法
// 目前不支持重排请求，直接返回错误
func (k *KieaiAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("Rerank not implemented for kieai channel")
}

// ConvertEmbeddingRequest 转换嵌入请求
// 实现channel.Adaptor接口的ConvertEmbeddingRequest方法
// 目前不支持嵌入请求，直接返回错误
func (k *KieaiAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("Embedding not implemented for kieai channel")
}

// ConvertAudioRequest 转换音频请求
// 实现channel.Adaptor接口的ConvertAudioRequest方法
// 目前不支持音频请求，直接返回错误
func (k *KieaiAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("Audio not implemented for kieai channel")
}

// ConvertImageRequest 转换图片生成请求
// 实现channel.Adaptor接口的ConvertImageRequest方法
// 目前不支持图片生成请求，直接返回错误
func (k *KieaiAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("Image not implemented for kieai channel")
}

// ConvertOpenAIResponsesRequest 转换OpenAI响应请求
// 实现channel.Adaptor接口的ConvertOpenAIResponsesRequest方法
// 目前不支持此功能，直接返回错误
func (k *KieaiAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("OpenAIResponses not implemented for kieai channel")
}

// DoRequest 执行请求
// 实现channel.Adaptor接口的DoRequest方法
// 接收gin.Context、relaycommon.RelayInfo和io.Reader参数
// 执行HTTP请求并返回响应
func (k *KieaiAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[KIEAI] DoRequest called with RequestURLPath: %s", info.RequestURLPath)

	// 读取并记录请求体
	var requestBodyCopy io.Reader
	var bodyBytes []byte
	if requestBody != nil {
		bodyBytes, _ = io.ReadAll(requestBody)
		logger.LogDebug(ctx, "[KIEAI] Original request body: %s", string(bodyBytes))
		requestBodyCopy = io.NopCloser(bytes.NewBuffer(bodyBytes))
	} else {
		requestBodyCopy = nil
	}

	// 所有请求统一使用GetRequestURL获取正确的URL
	fullURL, err := k.GetRequestURL(info)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[KIEAI] GetRequestURL failed: %v", err))
		return nil, err
	}

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, requestBodyCopy)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[KIEAI] New request failed: %v", err))
		return nil, err
	}

	// 设置请求头
	if err := k.SetupRequestHeader(c, &req.Header, info); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[KIEAI] SetupRequestHeader failed: %v", err))
		return nil, err
	}

	// 发送请求
	client := service.GetHttpClient()
	logger.LogDebug(ctx, "[KIEAI] Sending request to URL: %s with method: %s", req.URL.String(), req.Method)
	logger.LogDebug(ctx, "[KIEAI] Request headers: %v", req.Header)
	logger.LogDebug(ctx, "[KIEAI] Request body: %s", string(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[KIEAI] Client Do failed: %v", err))
		return nil, err
	}

	// 不要关闭响应体，因为DoResponse方法需要使用它
	// defer resp.Body.Close()

	// 读取响应体以便记录日志
	respBody, _ := io.ReadAll(resp.Body)
	logger.LogDebug(ctx, "[KIEAI] Response status: %d", resp.StatusCode)
	logger.LogDebug(ctx, "[KIEAI] Response headers: %v", resp.Header)
	logger.LogDebug(ctx, "[KIEAI] Response body: %s", string(respBody))

	// 重置响应体，以便DoResponse方法使用
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	logger.LogDebug(ctx, "[KIEAI] Request completed successfully, returning *http.Response")
	return resp, nil
}

// DoResponse 处理响应
// 实现channel.Adaptor接口的DoResponse方法
// 接收gin.Context、http.Response和relaycommon.RelayInfo参数
// 处理响应并转换为OpenAI格式
// 支持视频生成响应处理
func (k *KieaiAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	ctx := c.Request.Context()

	// 处理响应
	var body []byte
	var httpResp = resp

	// 如果响应为空，直接返回空的usage信息和nil错误
	if resp == nil {
		logger.LogDebug(ctx, "[KIEAI] Response is nil, returning empty usage")
		return &dto.Usage{}, nil
	}
	// 读取响应体
	var readErr error
	body, readErr = io.ReadAll(httpResp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(errors.New("Failed to read response body"), types.ErrorCodeBadResponse, 500)
	}

	// 如果响应体为空，直接返回空的usage信息和nil错误
	if len(body) == 0 {
		logger.LogDebug(ctx, "[KIEAI] Response body is empty, returning empty usage")
		return &dto.Usage{}, nil
	}

	logger.LogDebug(ctx, "[KIEAI] Response body: %s", string(body))

	// 重置响应体，以便后续处理
	if httpResp != nil {
		httpResp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// 检查是否已经是OpenAI格式的响应（检查是否有choices字段）
	var openAIRespCheck struct {
		Choices []any          `json:"choices"`
		Usage   map[string]any `json:"usage"`
	}
	if json.Unmarshal(body, &openAIRespCheck) == nil && openAIRespCheck.Choices != nil {
		// 如果已经是OpenAI格式的响应，提取usage信息
		logger.LogDebug(ctx, "[KIEAI] Response is already in OpenAI format, extracting usage")

		// 从openAIRespCheck中提取usage信息
		var usage dto.Usage
		if openAIRespCheck.Usage != nil {
			// 提取prompt_tokens
			if promptTokens, ok := openAIRespCheck.Usage["prompt_tokens"].(float64); ok {
				usage.PromptTokens = int(promptTokens)
			}
			// 提取completion_tokens
			if completionTokens, ok := openAIRespCheck.Usage["completion_tokens"].(float64); ok {
				usage.CompletionTokens = int(completionTokens)
			}
			// 计算total_tokens
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		}

		return &usage, nil
	}

	// 从RequestURLPath中提取路径部分
	path := info.RequestURLPath
	if idx := strings.Index(path, "?"); idx > 0 {
		path = path[:idx]
	}

	// 根据请求路径处理响应
	switch {
	case strings.HasPrefix(path, "/v1/videos"):
		// 处理sora-2视频生成响应
		var kieaiResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				TaskId string `json:"taskId"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &kieaiResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] DoResponse failed to parse sora-2 response: %v, body: %s", err, string(body)))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
		}

		if kieaiResp.Code != 200 {
			return nil, types.NewErrorWithStatusCode(errors.New(kieaiResp.Msg), types.ErrorCodeBadResponse, 500)
		}

		logger.LogDebug(ctx, "[KIEAI] Sora-2 video task created with ID: %s", kieaiResp.Data.TaskId)

		// 立即轮询获取结果
		pollResult, pollErr := k.pollVideoResult(c, info, kieaiResp.Data.TaskId)
		if pollErr != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Polling video result failed: %v", pollErr))
			// 轮询失败时返回错误信息
			return nil, types.NewErrorWithStatusCode(pollErr, types.ErrorCodeInvalidRequest, 400)
		}

		// 将轮询结果转换为OpenAI格式响应
		if httpResp != nil {
			openAIRespBody, _ := json.Marshal(pollResult)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))
		}

		return &dto.Usage{}, nil

	default:
		// 其他请求返回空的usage信息
		logger.LogDebug(ctx, "[KIEAI] Default response handling for URL path: %s", info.RequestURLPath)
		return &dto.Usage{}, nil
	}
}

// pollVideoResult 轮询视频生成结果
func (k *KieaiAdaptor) pollVideoResult(c *gin.Context, info *relaycommon.RelayInfo, taskId string) (*dto.OpenAIVideo, error) {
	ctx := c.Request.Context()
	logger.LogDebug(ctx, "[KIEAI] Polling video result for task ID: %s", taskId)

	// 轮询参数
	maxAttempts := 30
	attempts := 0
	interval := time.Second * 5

	// 轮询循环
	for attempts < maxAttempts {
		attempts++
		logger.LogDebug(ctx, "[KIEAI] Poll attempt %d/%d for task ID: %s", attempts, maxAttempts, taskId)

		// 构建轮询URL
		pollURL := "https://api.kie.ai/api/v1/jobs/recordInfo?taskId=" + taskId

		// 创建请求
		req, err := http.NewRequest("GET", pollURL, nil)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Creating poll request failed: %v", err))
			return nil, err
		}

		// 设置请求头
		req.Header.Set("Authorization", "Bearer "+k.apiKey)

		// 发送请求
		client := service.GetHttpClient()
		resp, err := client.Do(req)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Poll request failed: %v", err))
			// 轮询失败，继续下一次尝试
			time.Sleep(interval)
			continue
		}

		// 读取响应体
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Reading poll response failed: %v", err))
			// 轮询失败，继续下一次尝试
			time.Sleep(interval)
			continue
		}

		logger.LogDebug(ctx, "[KIEAI] Poll response: %s", string(respBody))

		// 解析响应
		var pollResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				TaskId       string  `json:"taskId"`
				Model        string  `json:"model"`
				State        string  `json:"state"`
				Param        string  `json:"param"`
				ResultJson   string  `json:"resultJson"`
				FailCode     *string `json:"failCode"`
				FailMsg      *string `json:"failMsg"`
				CostTime     int64   `json:"costTime"`
				CompleteTime int64   `json:"completeTime"`
				CreateTime   int64   `json:"createTime"`
			} `json:"data"`
		}

		if err := json.Unmarshal(respBody, &pollResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Parsing poll response failed: %v", err))
			// 轮询失败，继续下一次尝试
			time.Sleep(interval)
			continue
		}

		if pollResp.Code != 200 {
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Poll request returned error: %s", pollResp.Msg))
			// 轮询失败，继续下一次尝试
			time.Sleep(interval)
			continue
		}

		// 检查任务状态
		switch pollResp.Data.State {
		case "success":
			// 任务完成，解析结果
			logger.LogDebug(ctx, "[KIEAI] Video generation completed successfully")

			// 解析resultJson获取视频URL
			var resultJson struct {
				ResultUrls []string `json:"resultUrls"`
			}
			videoUrl := ""
			if err := json.Unmarshal([]byte(pollResp.Data.ResultJson), &resultJson); err == nil && len(resultJson.ResultUrls) > 0 {
				videoUrl = resultJson.ResultUrls[0]
				// 清理URL中的反引号和空格
				videoUrl = strings.Trim(videoUrl, " `")
			}

			// 解析param获取n_frames作为seconds
			seconds := "15" // 默认值
			if pollResp.Data.Param != "" {
				// 先解析param的外层结构
				var paramOuter struct {
					Input string `json:"input"`
				}
				if err := json.Unmarshal([]byte(pollResp.Data.Param), &paramOuter); err == nil {
					// 解析input字段
					var paramInput struct {
						NFrames string `json:"n_frames"`
					}
					if err := json.Unmarshal([]byte(paramOuter.Input), &paramInput); err == nil && paramInput.NFrames != "" {
						seconds = paramInput.NFrames
					}
				}
			}

			logger.LogDebug(ctx, "[KIEAI] Extracted video URL: %s, seconds: %s", videoUrl, seconds)

			// 构造OpenAI格式的视频响应
			videoResponse := &dto.OpenAIVideo{
				ID:        taskId,
				Status:    dto.VideoStatusCompleted,
				CreatedAt: pollResp.Data.CreateTime,
				Model:     "sora-2",
				Object:    "video",
				Seconds:   seconds,
				Progress:  100,
			}
			videoResponse.SetMetadata("url", videoUrl)

			return videoResponse, nil
		case "fail":
			// 任务失败，返回错误信息
			errorMsg := "Video generation failed"
			if pollResp.Data.FailMsg != nil && *pollResp.Data.FailMsg != "" {
				errorMsg = *pollResp.Data.FailMsg
			}
			if pollResp.Data.FailCode != nil && *pollResp.Data.FailCode != "" {
				errorMsg = fmt.Sprintf("%s (Code: %s)", errorMsg, *pollResp.Data.FailCode)
			}
			logger.LogError(ctx, fmt.Sprintf("[KIEAI] Video generation failed: %s", errorMsg))
			return nil, errors.New(errorMsg)
		case "waiting", "queuing", "generating":
			// 任务未完成，继续轮询
			logger.LogDebug(ctx, "[KIEAI] Video generation state: %s, continuing to poll", pollResp.Data.State)
			time.Sleep(interval)
			continue
		default:
			// 未知状态，继续轮询
			logger.LogDebug(ctx, "[KIEAI] Video generation state: %s, continuing to poll", pollResp.Data.State)
			time.Sleep(interval)
			continue
		}
	}

	// 轮询超时
	errorMsg := "Video generation polling timed out"
	logger.LogError(ctx, "[KIEAI] Video generation polling timed out")
	return nil, errors.New(errorMsg)
}

// GetModelList 获取支持的模型列表
// 实现channel.Adaptor接口的GetModelList方法
// 返回ModelList变量
func (k *KieaiAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 获取渠道名称
// 实现channel.Adaptor接口的GetChannelName方法
// 返回ChannelName常量
func (k *KieaiAdaptor) GetChannelName() string {
	return ChannelName
}

// ConvertClaudeRequest 转换Claude请求
// 实现channel.Adaptor接口的ConvertClaudeRequest方法
// 目前不支持Claude请求，直接返回错误
func (k *KieaiAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("ConvertClaudeRequest not implemented for kieai channel")
}

// ConvertGeminiRequest 转换Gemini请求
// 实现channel.Adaptor接口的ConvertGeminiRequest方法
// 目前不支持Gemini请求，直接返回错误
func (k *KieaiAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("ConvertGeminiRequest not implemented for kieai channel")
}

// ============================// TaskAdaptor Implementation// ============================

// ValidateRequestAndSetAction 验证请求并设置操作
// 实现channel.TaskAdaptor接口的ValidateRequestAndSetAction方法
func (t *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	// 验证请求并设置操作
	return relaycommon.ValidateMultipartDirect(c, info)
}

// BuildRequestURL 构建请求URL
// 实现channel.TaskAdaptor接口的BuildRequestURL方法
func (t *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 视频生成请求
	url := "https://api.kie.ai/api/v1/jobs/createTask"
	return url, nil
}

// BuildRequestHeader 构建请求头
// 实现channel.TaskAdaptor接口的BuildRequestHeader方法
func (t *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	// 设置请求头，需要Bearer拼接
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// BuildRequestBody 构建请求体
// 实现channel.TaskAdaptor接口的BuildRequestBody方法
func (t *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	// 获取任务请求
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	// 构建视频请求结构体
	var videoRequest struct {
		Model          string   `json:"model"`
		Prompt         string   `json:"prompt"`
		Seconds        string   `json:"seconds"`
		Size           string   `json:"size"`
		InputReference []string `json:"input_reference"`
	}

	// 从TaskSubmitReq中提取字段
	videoRequest.Model = req.Model
	videoRequest.Prompt = req.Prompt
	// 同时检查Seconds和Duration字段
	if req.Seconds != "" {
		videoRequest.Seconds = req.Seconds
	} else if req.Duration != 0 {
		videoRequest.Seconds = strconv.Itoa(req.Duration)
	}
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

	// 构建kieAi API请求体
	kieaiRequest, err := t.buildKieaiVideoRequest(videoRequest)
	if err != nil {
		return nil, err
	}

	// 转换为JSON
	requestBody, err := json.Marshal(kieaiRequest)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(requestBody), nil
}

// DoRequest 执行请求
// 实现channel.TaskAdaptor接口的DoRequest方法
func (t *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	// 使用通用的API请求方法
	return channel.DoTaskApiRequest(t, c, info, requestBody)
}

// DoResponse 处理响应
// 实现channel.TaskAdaptor接口的DoResponse方法
func (t *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	// 解析响应
	var kieaiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			TaskId string `json:"taskId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(responseBody, &kieaiResp); err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	// 检查响应代码
	if kieaiResp.Code != 200 {
		taskErr = service.TaskErrorWrapper(fmt.Errorf(kieaiResp.Msg), "api_request_failed", http.StatusInternalServerError)
		return
	}

	// 返回任务ID和响应数据
	return kieaiResp.Data.TaskId, responseBody, nil
}

// FetchTask 获取任务状态
// 实现channel.TaskAdaptor接口的FetchTask方法
func (t *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	// 构建轮询URL
	taskId, ok := body["task_id"].(string)
	if !ok {
		return nil, errors.New("task_id not found in body")
	}

	pollURL := "https://api.kie.ai/api/v1/jobs/recordInfo?taskId=" + taskId

	// 创建请求
	req, err := http.NewRequest("GET", pollURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := service.GetHttpClient()
	return client.Do(req)
}

// ParseTaskResult 解析任务结果
// 实现channel.TaskAdaptor接口的ParseTaskResult方法
func (t *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// 解析响应
	var pollResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			TaskId       string  `json:"taskId"`
			Model        string  `json:"model"`
			State        string  `json:"state"`
			Param        string  `json:"param"`
			ResultJson   string  `json:"resultJson"`
			FailCode     *string `json:"failCode"`
			FailMsg      *string `json:"failMsg"`
			CostTime     int64   `json:"costTime"`
			CompleteTime int64   `json:"completeTime"`
			CreateTime   int64   `json:"createTime"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &pollResp); err != nil {
		return nil, err
	}

	// 检查响应代码
	if pollResp.Code != 200 {
		return nil, errors.New(pollResp.Msg)
	}

	// 构建任务信息
	taskInfo := &relaycommon.TaskInfo{}

	// 根据状态设置相应信息
	switch pollResp.Data.State {
	case "success":
		// 任务成功完成，解析resultJson获取视频URL
		taskInfo.Status = "SUCCESS"

		// 解析resultJson获取视频URL
		var resultJson struct {
			ResultUrls []string `json:"resultUrls"`
		}
		if err := json.Unmarshal([]byte(pollResp.Data.ResultJson), &resultJson); err == nil && len(resultJson.ResultUrls) > 0 {
			taskInfo.Url = strings.Trim(resultJson.ResultUrls[0], " `")
		}
	case "fail":
		// 任务失败，设置错误信息
		taskInfo.Status = "FAILURE"
		if pollResp.Data.FailMsg != nil && *pollResp.Data.FailMsg != "" {
			taskInfo.Reason = *pollResp.Data.FailMsg
			if pollResp.Data.FailCode != nil && *pollResp.Data.FailCode != "" {
				taskInfo.Reason = fmt.Sprintf("%s (Code: %s)", taskInfo.Reason, *pollResp.Data.FailCode)
			}
		} else {
			taskInfo.Reason = "Video generation failed"
		}
	case "waiting", "queuing", "generating":
		// 任务正在处理中
		taskInfo.Status = "PENDING"
		taskInfo.Reason = "Task is still processing"
	default:
		// 未知状态
		taskInfo.Status = "PENDING"
		taskInfo.Reason = "Task is still processing"
	}

	return taskInfo, nil
}
