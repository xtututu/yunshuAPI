package suchuang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"xunkecloudAPI/common"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/logger"
	"xunkecloudAPI/model"
	"xunkecloudAPI/relay/channel"
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
	"nano-banana",
	"nano-banana-pro",
	"nano-banana-pro_1k",
	"nano-banana-pro_2k",
	"nano-banana-pro_4k",
	"gemini-3-pro",
	"gemini-2.5-pro",
	"gemini-3-pro-preview",
	"sora-2",
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

// TaskAdaptor 实现channel.TaskAdaptor接口的适配器类型
type TaskAdaptor struct {
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
		modelName := info.OriginModelName
		upstreamModelName := ""
		if info.ChannelMeta != nil && info.ChannelMeta.UpstreamModelName != "" {
			upstreamModelName = info.ChannelMeta.UpstreamModelName
		} else if info.UpstreamModelName != "" {
			upstreamModelName = info.UpstreamModelName
		}
		if upstreamModelName != "" {
			fmt.Printf("[SUCHUANG] GetRequestURL called with: baseURL=%s, RequestURLPath=%s, Model=%s, UpstreamModel=%s\n", s.baseURL, info.RequestURLPath, modelName, upstreamModelName)
		} else {
			fmt.Printf("[SUCHUANG] GetRequestURL called with: baseURL=%s, RequestURLPath=%s, Model=%s\n", s.baseURL, info.RequestURLPath, modelName)
		}
	}
	// 如果是图片生成请求，根据模型返回不同的API端点
	if info.RequestURLPath == "/v1/images/generations" {
		// 确定当前实际使用的模型名称（优先使用映射后的模型名称）
		currentModelName := info.OriginModelName
		if info.ChannelMeta != nil && info.ChannelMeta.UpstreamModelName != "" {
			currentModelName = info.ChannelMeta.UpstreamModelName
		} else if info.UpstreamModelName != "" {
			currentModelName = info.UpstreamModelName
		}

		// 为nano-banana模型使用专用API接口
		if currentModelName == "nano-banana" {
			url := "https://api.wuyinkeji.com/api/img/nanoBanana"
			if common.DebugEnabled {
				fmt.Printf("[SUCHUANG] Generated image generation URL for nano-banana: %s\n", url)
			}
			return url, nil
		}
		// 所有其他nano-banana模型变体使用统一的API接口
		if strings.HasPrefix(currentModelName, "nano-banana-pro") {
			url := "https://api.wuyinkeji.com/api/img/nanoBanana-pro"
			if common.DebugEnabled {
				fmt.Printf("[SUCHUANG] Generated image generation URL for %s: %s\n", currentModelName, url)
			}
			return url, nil
		}
		// 其他图片模型使用默认接口
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
	// 如果是视频生成请求，根据模型返回不同的API端点
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		// 根据模型选择不同的API端点
		url := "https://api.wuyinkeji.com/api/sora2/submit"
		if info.UpstreamModelName == "sora-2-pro" {
			url = "https://api.wuyinkeji.com/api/sora2pro/submit"
		}
		if common.DebugEnabled {
			fmt.Printf("[SUCHUANG] Generated video generation URL: %s for model: %s\n", url, info.UpstreamModelName)
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
// 支持图片生成请求、Gemini-3-Pro视频内容识别请求和sora-2视频生成请求转换
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

	// 支持sora-2视频生成请求
	if strings.HasPrefix(info.RequestURLPath, "/v1/videos") {
		logger.LogDebug(ctx, "[SUCHUANG] Converting video generation request for %s", request.Model)

		// 定义视频请求结构体
		var videoRequest struct {
			Model          string   `json:"model"`
			Prompt         string   `json:"prompt"`
			Seconds        string   `json:"seconds"`
			Size           string   `json:"size"`
			InputReference []string `json:"input_reference"`
		}

		// 将GeneralOpenAIRequest转换为map以便获取所有字段，包括视频生成特有的字段
		requestMap := request.ToMap()

		// 从requestMap中提取所有字段
		if model, ok := requestMap["model"].(string); ok {
			videoRequest.Model = model
		} else {
			// 从request对象获取model
			videoRequest.Model = request.Model
		}

		if prompt, ok := requestMap["prompt"].(string); ok {
			videoRequest.Prompt = prompt
		} else if request.Prompt != nil {
			// 从request对象获取prompt
			if promptStr, ok := request.Prompt.(string); ok {
				videoRequest.Prompt = promptStr
			}
		}

		// 处理Seconds字段，支持多种类型
		if seconds, ok := requestMap["seconds"].(string); ok {
			videoRequest.Seconds = seconds
		} else if secondsFloat, ok := requestMap["seconds"].(float64); ok {
			videoRequest.Seconds = fmt.Sprintf("%.0f", secondsFloat)
		} else if secondsInt, ok := requestMap["seconds"].(int); ok {
			videoRequest.Seconds = fmt.Sprintf("%d", secondsInt)
		} else {
			// 默认值
			videoRequest.Seconds = "15"
		}

		if size, ok := requestMap["size"].(string); ok {
			videoRequest.Size = size
		} else {
			// 默认值
			videoRequest.Size = "1024×1792"
		}

		// 处理InputReference字段
		if inputReference, ok := requestMap["input_reference"].([]string); ok {
			videoRequest.InputReference = inputReference
		} else if inputReferenceInterface, ok := requestMap["input_reference"].([]any); ok {
			for _, ref := range inputReferenceInterface {
				if refStr, ok := ref.(string); ok {
					videoRequest.InputReference = append(videoRequest.InputReference, refStr)
				}
			}
		}

		// 确保使用正确的模型
		if videoRequest.Model == "" {
			videoRequest.Model = request.Model
		}

		// 设置默认值
		if videoRequest.Seconds == "" {
			videoRequest.Seconds = "15" // 默认15秒
		}

		// 转换size到aspectRatio
		aspectRatio := "16:9" // 默认16:9
		if videoRequest.Size == "720×1280" || videoRequest.Size == "720x1280" || videoRequest.Size == "1024×1792" || videoRequest.Size == "1024x1792" {
			aspectRatio = "9:16"
		}

		// 获取input_reference中的URL，默认取第一个
		url := ""
		if len(videoRequest.InputReference) > 0 {
			url = strings.Trim(videoRequest.InputReference[0], " `")
		}

		// 构建sora-2 API请求体
		// 将seconds转换为数字类型
		var duration int
		if secondsStr := videoRequest.Seconds; secondsStr != "" {
			if d, err := strconv.Atoi(secondsStr); err == nil {
				duration = d
			} else {
				// 默认15秒
				duration = 15
			}
		} else {
			// 默认15秒
			duration = 15
		}

		// 根据模型类型构建不同的请求体
		soraRequest := map[string]interface{}{
			"prompt":      videoRequest.Prompt,
			"duration":    duration,
			"aspectRatio": aspectRatio,
			"url":         url,
		}

		// 只有当模型不是sora-2-pro时，才添加size字段
		if videoRequest.Model != "sora-2-pro" && info.UpstreamModelName != "sora-2-pro" {
			soraRequest["size"] = "auto"
		}

		requestData, _ := json.Marshal(soraRequest)
		logger.LogDebug(ctx, "[SUCHUANG] Generated sora-2 video request body: %s", string(requestData))
		return soraRequest, nil
	}

	// 支持视频内容识别请求（支持gemini-3-pro和gpt-5模型）
	if info.RequestURLPath == "/v1/chat/completions" || info.RequestURLPath == "/plus/v1/chat/completions" {
		logger.LogDebug(ctx, "[SUCHUANG] Converting chat completion request for %s", request.Model)

		// 当渠道是速创时，如果模型是gemini-3-pro-preview则强制修改为gemini-3-pro，适用于所有chat/completions接口
		if request.Model == "gemini-3-pro-preview" {
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
	// 确定当前实际使用的模型名称（优先使用映射后的模型名称）
	currentModelName := info.OriginModelName
	upstreamModelName := ""
	if info.ChannelMeta != nil && info.ChannelMeta.UpstreamModelName != "" {
		upstreamModelName = info.ChannelMeta.UpstreamModelName
		currentModelName = upstreamModelName
	} else if info.UpstreamModelName != "" {
		upstreamModelName = info.UpstreamModelName
		currentModelName = upstreamModelName
	}

	if upstreamModelName != "" {
		logger.LogDebug(ctx, "[SUCHUANG] ConvertImageRequest called with: Model=%s, UpstreamModel=%s, Prompt=%s", request.Model, upstreamModelName, request.Prompt)
	} else {
		logger.LogDebug(ctx, "[SUCHUANG] ConvertImageRequest called with: Model=%s, Prompt=%s", request.Model, request.Prompt)
	}
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

	// 为nano-banana模型使用专用入参结构
	if currentModelName == "nano-banana" {
		// 构建nano-banana模型的速创API请求体
		suchuangRequest := map[string]interface{}{
			"model":       "nano-banana",
			"prompt":      request.Prompt,
			"aspectRatio": "auto", // 默认值
			"img_url":     imageUrls,
		}

		requestData, _ := json.Marshal(suchuangRequest)
		logger.LogDebug(ctx, "[SUCHUANG] Generated request body for nano-banana: %s", string(requestData))
		return suchuangRequest, nil
	}

	// 为nano-banana-pro模型变体构建请求体
	if strings.HasPrefix(currentModelName, "nano-banana-pro") {
		// 确定当前使用的模型名称
		modelName := currentModelName

		// 根据模型名称设置正确的imageSize
		imageSize := "1K" // 默认值，对应nano-banana-pro
		re := regexp.MustCompile(`nano-banana-pro_(\d+)k`)
		if matches := re.FindStringSubmatch(strings.ToLower(modelName)); len(matches) == 2 {
			// 提取尺寸数字并大写K
			imageSize = matches[1] + "K"
		}

		// 构建速创API请求体
		suchuangRequest := map[string]interface{}{
			"prompt":      request.Prompt,
			"aspectRatio": "auto",
			"imageSize":   imageSize,
			"img_url":     imageUrls,
		}

		requestData, _ := json.Marshal(suchuangRequest)
		logger.LogDebug(ctx, "[SUCHUANG] Generated request body for %s with imageSize=%s: %s", modelName, imageSize, string(requestData))
		return suchuangRequest, nil
	}

	// 其他模型使用默认入参结构
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
	client := service.GetHttpClient()
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

	// 如果响应为空，直接返回空的usage信息和nil错误
	if resp == nil {
		logger.LogDebug(ctx, "[SUCHUANG] Response is nil, returning empty usage")
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
		Choices []any          `json:"choices"`
		Usage   map[string]any `json:"usage"`
	}
	if json.Unmarshal(body, &openAIRespCheck) == nil && openAIRespCheck.Choices != nil {
		// 如果已经是OpenAI格式的响应，提取usage信息
		logger.LogDebug(ctx, "[SUCHUANG] Response is already in OpenAI format, extracting usage")

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

	// 从RequestURLPath中提取路径部分，因为它包含完整URL
	path := info.RequestURLPath
	if idx := strings.Index(path, "?"); idx > 0 {
		path = path[:idx]
	}

	// 根据请求路径处理响应
	switch {
	case strings.HasPrefix(path, "/v1/videos"):
		// 处理sora-2视频生成响应
		var soraResp struct {
			Msg  string `json:"msg"`
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
			Code     int     `json:"code"`
			ExecTime float64 `json:"exec_time,omitempty"`
			IP       string  `json:"ip,omitempty"`
		}

		if err := json.Unmarshal(body, &soraResp); err != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] DoResponse failed to parse sora-2 response: %v, body: %s", err, string(body)))
			return nil, types.NewErrorWithStatusCode(errors.New("Failed to parse response"), types.ErrorCodeBadResponse, 500)
		}

		if soraResp.Code != 200 {
			return nil, types.NewErrorWithStatusCode(errors.New(soraResp.Msg), types.ErrorCodeBadResponse, 500)
		}

		logger.LogDebug(ctx, "[SUCHUANG] Sora-2 video task created with ID: %s", soraResp.Data.ID)

		// 构造视频生成响应，返回任务ID
		// 使用默认值10秒，因为在DoResponse方法中无法直接访问原始请求
		seconds := "10"

		// 使用与其他视频API一致的OpenAIVideo格式
		videoResponse := &dto.OpenAIVideo{
			ID:        fmt.Sprintf("%s", soraResp.Data.ID),
			Status:    dto.VideoStatusQueued,
			CreatedAt: time.Now().UnixMilli(),
			Model:     "sora-2",
			Object:    "video",
			Seconds:   seconds,
			Progress:  0,
		}

		// 将转换后的响应写回响应体
		if httpResp != nil {
			videoRespBody, _ := json.Marshal(videoResponse)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(videoRespBody))
		}

		return &dto.Usage{}, nil
	case strings.HasPrefix(path, "/v1/images/generations"):
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

		logger.LogDebug(ctx, "[SUCHUANG] Image task created with ID: %d", createTaskResp.Data.ID)

		// 立即轮询获取结果
		pollResult, pollErr := s.pollImageResult(c, info, createTaskResp.Data.ID)
		if pollErr != nil {
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Polling image result failed: %v", pollErr))
			// 轮询失败时返回空URL，让用户稍后手动查询
			openAIResp := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"url":            "",
						"revised_prompt": "",
					},
				},
				"created": time.Now().Unix(),
			}
			if httpResp != nil {
				openAIRespBody, _ := json.Marshal(openAIResp)
				httpResp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))
			}
			return &dto.Usage{}, nil
		}

		// 将轮询结果转换为OpenAI格式响应
		if httpResp != nil {
			openAIRespBody, _ := json.Marshal(pollResult)
			httpResp.Body = io.NopCloser(bytes.NewBuffer(openAIRespBody))
		}

		return &dto.Usage{}, nil

	case strings.HasPrefix(path, "/api/img/drawDetail"):
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
		}

		// 返回空的usage信息和nil错误
		return &dto.Usage{}, nil

	case strings.HasPrefix(path, "/v1/chat/completions"):
		fallthrough
	case strings.HasPrefix(path, "/plus/v1/chat/completions"):
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
					ID:      fmt.Sprintf("chatcmpl-%s", time.Now().Format("20060102150405106725986")+"QLKsygEy"),
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
						"prompt_tokens":     9,
						"completion_tokens": 260,
						"total_tokens":      269,
						"prompt_tokens_details": map[string]any{
							"text_tokens":           9,
							"cached_tokens_details": map[string]any{},
						},
						"completion_tokens_details": map[string]any{
							"reasoning_tokens": 249,
						},
						"claude_cache_creation_5_m_tokens": 0,
						"claude_cache_creation_1_h_tokens": 0,
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

			// 从openAIResp中提取usage信息并返回
			var usage dto.Usage
			if openAIResp.Usage != nil {
				// 提取prompt_tokens
				if promptTokens, ok := openAIResp.Usage["prompt_tokens"].(float64); ok {
					usage.PromptTokens = int(promptTokens)
				}
				// 提取completion_tokens
				if completionTokens, ok := openAIResp.Usage["completion_tokens"].(float64); ok {
					usage.CompletionTokens = int(completionTokens)
				}
				// 计算total_tokens
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}

			return &usage, nil
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
				ID:      fmt.Sprintf("chatcmpl-%s", time.Now().Format("20060102150405106725986")+"QLKsygEy"),
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
					"prompt_tokens":     9,
					"completion_tokens": 260,
					"total_tokens":      269,
					"prompt_tokens_details": map[string]any{
						"text_tokens":           9,
						"cached_tokens_details": map[string]any{},
					},
					"completion_tokens_details": map[string]any{
						"reasoning_tokens": 249,
					},
					"claude_cache_creation_5_m_tokens": 0,
					"claude_cache_creation_1_h_tokens": 0,
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

		// 从openAIResp中提取usage信息并返回
		var usage dto.Usage
		if openAIResp.Usage != nil {
			// 提取prompt_tokens
			if promptTokens, ok := openAIResp.Usage["prompt_tokens"].(float64); ok {
				usage.PromptTokens = int(promptTokens)
			}
			// 提取completion_tokens
			if completionTokens, ok := openAIResp.Usage["completion_tokens"].(float64); ok {
				usage.CompletionTokens = int(completionTokens)
			}
			// 计算total_tokens
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		}

		return &usage, nil

	default:
		// 其他请求返回空的usage信息
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
	// 根据模型选择不同的API端点
	url := "https://api.wuyinkeji.com/api/sora2/submit"
	if info.UpstreamModelName == "sora-2-pro" {
		url = "https://api.wuyinkeji.com/api/sora2pro/submit"
	}
	return url, nil
}

// BuildRequestHeader 构建请求头
// 实现channel.TaskAdaptor接口的BuildRequestHeader方法
func (t *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	// 设置请求头，不需要Bearer拼接
	req.Header.Set("Authorization", t.apiKey)
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

	// 转换size到aspectRatio
	aspectRatio := "16:9" // 默认16:9
	if videoRequest.Size == "720×1280" || videoRequest.Size == "720x1280" || videoRequest.Size == "1024×1792" || videoRequest.Size == "1024x1792" {
		aspectRatio = "9:16"
	}

	// 获取input_reference中的URL，默认取第一个
	url := ""
	if len(videoRequest.InputReference) > 0 {
		url = strings.Trim(videoRequest.InputReference[0], " `")
	}

	// 构建速创API请求体
	// 将seconds转换为数字类型
	var duration int
	if secondsStr := videoRequest.Seconds; secondsStr != "" {
		if d, err := strconv.Atoi(secondsStr); err == nil {
			duration = d
		} else {
			// 默认15秒
			duration = 15
		}
	} else {
		// 默认15秒
		duration = 15
	}

	// 根据模型类型构建不同的请求体
	soraRequest := map[string]interface{}{
		"prompt":      videoRequest.Prompt,
		"duration":    duration,
		"aspectRatio": aspectRatio,
		"url":         url,
	}

	// 只有当模型不是sora-2-pro时，才添加size字段
	if videoRequest.Model != "sora-2-pro" && info.UpstreamModelName != "sora-2-pro" {
		soraRequest["size"] = "auto"
	}

	// 转换为JSON
	requestBody, err := json.Marshal(soraRequest)
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
	var suchuangResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(responseBody, &suchuangResp); err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	// 检查响应代码
	if suchuangResp.Code != 200 {
		taskErr = service.TaskErrorWrapper(fmt.Errorf(suchuangResp.Msg), "api_request_failed", http.StatusInternalServerError)
		return
	}

	// 返回任务ID和响应数据
	return suchuangResp.Data.ID, responseBody, nil
}

// FetchTask 获取任务
// 实现channel.TaskAdaptor接口的FetchTask方法
func (t *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	// 从body中获取task_id
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	// 构建轮询URL
	url := fmt.Sprintf("https://api.wuyinkeji.com/api/sora2/detail?id=%s", taskID)

	// 创建GET请求
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", key)

	// 发送请求
	return service.GetHttpClient().Do(req)
}

// ParseTaskResult 解析任务结果
// 实现channel.TaskAdaptor接口的ParseTaskResult方法
func (t *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// 解析响应
	var suchuangResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Content     string `json:"content"`
			Status      int    `json:"status"`
			FailReason  string `json:"fail_reason"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
			RemoteURL   string `json:"remote_url"`
			Size        string `json:"size"`
			Duration    int    `json:"duration"`
			AspectRatio string `json:"aspectRatio"`
			URL         string `json:"url"`
			PID         string `json:"pid"`
			ID          string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &suchuangResp); err != nil {
		return nil, err
	}

	// 转换状态和进度
	var status model.TaskStatus
	var progress int
	switch suchuangResp.Data.Status {
	case 0: // 排队中
		status = model.TaskStatusQueued
		progress = 0
	case 1: // 成功
		status = model.TaskStatusSuccess
		progress = 100
	case 2: // 失败
		status = model.TaskStatusFailure
		progress = 0
	case 3: // 生成中
		status = model.TaskStatusInProgress
		progress = 50
	default: // 其他状态默认处理中
		status = model.TaskStatusInProgress
		progress = 0
	}

	// 构建任务结果
	taskResult := &relaycommon.TaskInfo{
		Code:     0,
		TaskID:   suchuangResp.Data.ID,
		Status:   string(status),
		Progress: fmt.Sprintf("%d%%", progress),
	}

	// 如果任务成功，设置URL
	if status == model.TaskStatusSuccess {
		taskResult.Url = suchuangResp.Data.RemoteURL
		taskResult.RemoteUrl = suchuangResp.Data.RemoteURL
	}

	// 如果任务失败，设置失败原因
	if status == model.TaskStatusFailure {
		taskResult.Reason = suchuangResp.Data.FailReason
	}

	return taskResult, nil
}

// GetModelList 获取支持的模型列表
// 实现channel.TaskAdaptor接口的GetModelList方法
func (t *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName 获取渠道名称
// 实现channel.TaskAdaptor接口的GetChannelName方法
func (t *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// UserExpectedVideoResponse 用户期望的视频响应格式
// 与其他视频API保持一致
type UserExpectedVideoResponse struct {
	ID        string `json:"id"`
	Size      string `json:"size"`
	Model     string `json:"model"`
	Object    string `json:"object"`
	Status    string `json:"status"`
	Seconds   string `json:"seconds"`
	Progress  int    `json:"progress"`
	VideoURL  string `json:"video_url"`
	CreatedAt int64  `json:"created_at"`
}

// ConvertToOpenAIVideo 转换任务为OpenAI格式的视频响应
// 实现channel.OpenAIVideoConverter接口的ConvertToOpenAIVideo方法
func (t *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	// 解析任务数据，获取原始请求的seconds和size
	var taskData struct {
		Seconds string `json:"seconds"`
		Size    string `json:"size"`
	}
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		// 如果解析失败，使用默认值
		taskData.Seconds = "10"
		taskData.Size = "1024×1792"
	}

	// 构建用户期望的视频响应格式
	response := UserExpectedVideoResponse{
		ID:        task.TaskID,
		Size:      taskData.Size,
		Model:     "sora-2",
		Object:    "video",
		Status:    task.Status.ToVideoStatus(),
		Seconds:   taskData.Seconds,
		Progress:  0,
		CreatedAt: task.CreatedAt,
	}

	// 根据任务状态设置进度和视频URL
	switch task.Status {
	case model.TaskStatusSuccess:
		response.Progress = 100
		response.VideoURL = task.FailReason // 使用FailReason字段存储视频URL
	case model.TaskStatusInProgress:
		response.Progress = 50
	case model.TaskStatusQueued, model.TaskStatusSubmitted:
		response.Progress = 0
	case model.TaskStatusFailure:
		response.Progress = 0
	}

	// 序列化响应为JSON
	return json.Marshal(response)
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
	client := service.GetHttpClient()
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

	// 构造轮询URL（使用GET方法，参数通过查询字符串传递）
	pollURL := fmt.Sprintf("%s/api/img/drawDetail?id=%d", s.baseURL, taskID)

	// 设置轮询间隔和超时
	pollInterval := 2 * time.Second
	timeout := 900 * time.Second
	startTime := time.Now()

	// 轮询获取结果
	for {
		// 创建GET请求
		req, err := http.NewRequest("GET", pollURL, nil)
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
		client := service.GetHttpClient()
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

		// 检查任务是否失败（status=3）
		if pollResult.Data.Status == 3 {
			// 任务失败，返回失败原因
			failReason := pollResult.Data.FailReason
			if failReason == "" {
				failReason = "Image generation failed"
			}
			logger.LogError(ctx, fmt.Sprintf("[SUCHUANG] Image generation failed: %s", failReason))
			return nil, errors.New(failReason)
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
