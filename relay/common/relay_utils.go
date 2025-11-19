package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"yishangyunApi/common"
	"yishangyunApi/constant"
	"yishangyunApi/dto"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type HasPrompt interface {
	GetPrompt() string
}

type HasImage interface {
	HasImage() bool
}

func GetFullRequestURL(baseURL string, requestURL string, channelType int) string {
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case constant.ChannelTypeOpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case constant.ChannelTypeAzure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}

func GetAPIVersion(c *gin.Context) string {
	query := c.Request.URL.Query()
	apiVersion := query.Get("api-version")
	if apiVersion == "" {
		apiVersion = c.GetString("api_version")
	}
	return apiVersion
}

func createTaskError(err error, code string, statusCode int, localError bool) *dto.TaskError {
	return &dto.TaskError{
		Code:       code,
		Message:    err.Error(),
		StatusCode: statusCode,
		LocalError: localError,
		Error:      err,
	}
}

func storeTaskRequest(c *gin.Context, info *RelayInfo, action string, requestObj TaskSubmitReq) {
	info.Action = action
	c.Set("task_request", requestObj)
}
func GetTaskRequest(c *gin.Context) (TaskSubmitReq, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return TaskSubmitReq{}, fmt.Errorf("request not found in context")
	}
	req, ok := v.(TaskSubmitReq)
	if !ok {
		return TaskSubmitReq{}, fmt.Errorf("invalid task request type")
	}
	return req, nil
}

func validatePrompt(prompt string) *dto.TaskError {
	if strings.TrimSpace(prompt) == "" {
		return createTaskError(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest, true)
	}
	return nil
}

// getFormDataValue 从 form data 中获取值，处理字符串切片
func getFormDataValue(formData map[string][]string, key string) string {
	if values, exists := formData[key]; exists && len(values) > 0 {
		return values[0]
	}
	return ""
}

func validateMultipartTaskRequest(c *gin.Context, info *RelayInfo, action string) (TaskSubmitReq, error) {
	var req TaskSubmitReq

	// 使用自定义的 multipart 解析函数，避免 Gin 内部状态问题
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return req, fmt.Errorf("multipart form error: %w", err)
	}

	formData := form.Value
	req = TaskSubmitReq{
		Prompt:   getFormDataValue(formData, "prompt"),
		Model:    getFormDataValue(formData, "model"),
		Mode:     getFormDataValue(formData, "mode"),
		Image:    getFormDataValue(formData, "image"),
		Size:     getFormDataValue(formData, "size"),
		Metadata: make(map[string]interface{}),
	}

	// 验证 prompt 字段
	if taskErr := validatePrompt(req.Prompt); taskErr != nil {
		return req, fmt.Errorf("prompt validation failed: %s", taskErr.Message)
	}

	if durationStr := getFormDataValue(formData, "seconds"); durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil {
			req.Duration = duration
		}
	}

	if images := formData["images"]; len(images) > 0 {
		req.Images = images
	}

	// 处理 input_reference 字段
	if inputRefs := formData["input_reference"]; len(inputRefs) > 0 {
		inputRefValue := inputRefs[0]
		// 尝试解析为 JSON
		var parsed interface{}
		if err := json.Unmarshal([]byte(inputRefValue), &parsed); err == nil {
			// 成功解析为 JSON
			switch v := parsed.(type) {
			case string:
				req.InputReference = v
			case []interface{}:
				strSlice := make([]string, len(v))
				for i, item := range v {
					if str, ok := item.(string); ok {
						strSlice[i] = str
					}
				}
				req.InputReference = strSlice
			default:
				req.InputReference = inputRefValue
			}
		} else {
			// 不是 JSON，作为普通字符串处理
			if len(inputRefs) == 1 {
				req.InputReference = inputRefs[0] // 单个值作为字符串
			} else {
				req.InputReference = inputRefs // 多个值作为字符串数组
			}
		}
	}

	for key, values := range formData {
		if len(values) > 0 && !isKnownTaskField(key) {
			if intVal, err := strconv.Atoi(values[0]); err == nil {
				req.Metadata[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(values[0], 64); err == nil {
				req.Metadata[key] = floatVal
			} else {
				req.Metadata[key] = values[0]
			}
		}
	}
	return req, nil
}

func ValidateMultipartDirect(c *gin.Context, info *RelayInfo) *dto.TaskError {
	var prompt string
	var model string
	var seconds int
	var size string
	var hasInputReference bool

	// 先尝试获取 model 参数来判断是否需要转换
	contentType := c.GetHeader("Content-Type")

	// 如果是 JSON 请求，先尝试解析获取 model
	if strings.HasPrefix(contentType, "application/json") {
		var tempReq struct {
			Model string `json:"model"`
		}
		body, err := common.GetRequestBody(c)
		if err == nil {
			json.Unmarshal(body, &tempReq)

			// 检查是否需要转换为 multipart
			if shouldConvertToMultipart(tempReq.Model, contentType) {
				// 直接使用已经获取的 body 进行转换，避免重复读取
				if err := convertJSONToMultipartWithBody(c, body); err != nil {
					return createTaskError(err, "conversion_failed", http.StatusBadRequest, true)
				}
				// 转换后重新获取 Content-Type
				contentType = c.GetHeader("Content-Type")
			}
		}
	}

	var req TaskSubmitReq
	var err error

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// 手动解析 multipart 数据，避免 interface{} 类型绑定问题
		req, err = validateMultipartTaskRequest(c, info, constant.TaskActionGenerate)
		if err != nil {
			return createTaskError(err, "invalid_multipart_form", http.StatusBadRequest, true)
		}
	} else {
		// 对于 JSON，使用 UnmarshalBodyReusable
		if err := common.UnmarshalBodyReusable(c, &req); err != nil {
			return createTaskError(err, "invalid_request_body", http.StatusBadRequest, true)
		}
	}

	if err != nil {
		return createTaskError(err, "invalid_request_body", http.StatusBadRequest, true)
	}

	prompt = req.Prompt
	model = req.Model
	size = req.Size
	seconds, _ = strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}

	// 处理 input_reference 字段
	if req.InputReference != nil {
		fmt.Printf("DEBUG: InputReference type: %T, value: %v\n", req.InputReference, req.InputReference)
		switch v := req.InputReference.(type) {
		case string:
			fmt.Printf("DEBUG: Processing as string: %s\n", v)
			if v != "" {
				req.Images = []string{v}
				hasInputReference = true
			}
		case []string:
			fmt.Printf("DEBUG: Processing as []string: %v\n", v)
			if len(v) > 0 {
				req.Images = v
				hasInputReference = true
			}
		case []interface{}:
			fmt.Printf("DEBUG: Processing as []interface{}: %v\n", v)
			strSlice := make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					strSlice[i] = str
				}
			}
			if len(strSlice) > 0 {
				req.Images = strSlice
				hasInputReference = true
			}
		default:
			fmt.Printf("DEBUG: Unknown type for InputReference: %T, value: %v\n", v, v)
		}
	}

	if req.HasImage() {
		hasInputReference = true
	}

	// 确保 model 字段存在且不为空
	if strings.TrimSpace(model) == "" {
		// 尝试从请求中重新获取 model
		if strings.HasPrefix(contentType, "multipart/form-data") {
			if modelValue := c.PostForm("model"); modelValue != "" {
				model = modelValue
			}
		}
	}

	if strings.TrimSpace(model) == "" {
		return createTaskError(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest, true)
	}

	// 确保 prompt 字段存在且不为空
	if strings.TrimSpace(prompt) == "" {
		// 尝试从请求中重新获取 prompt
		if strings.HasPrefix(contentType, "multipart/form-data") {
			if promptValue := c.PostForm("prompt"); promptValue != "" {
				prompt = promptValue
			}
		}
	}

	if taskErr := validatePrompt(prompt); taskErr != nil {
		return taskErr
	}

	action := constant.TaskActionTextGenerate
	if hasInputReference {
		action = constant.TaskActionGenerate
	}
	if strings.HasPrefix(model, "sora-2") {

		if size == "" {
			size = "720x1280"
		}

		if seconds <= 0 {
			seconds = 4
		}

		if model == "sora-2" && !lo.Contains([]string{"720x1280", "1280x720"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		if model == "sora-2-pro" && !lo.Contains([]string{"720x1280", "1280x720", "1792x1024", "1024x1792"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		info.PriceData.OtherRatios = map[string]float64{
			"seconds": float64(seconds),
			"size":    1,
		}
		if lo.Contains([]string{"1792x1024", "1024x1792"}, size) {
			info.PriceData.OtherRatios["size"] = 1.666667
		}
	}

	info.Action = action

	return nil
}

func isKnownTaskField(field string) bool {
	knownFields := map[string]bool{
		"prompt":          true,
		"model":           true,
		"mode":            true,
		"image":           true,
		"images":          true,
		"size":            true,
		"duration":        true,
		"input_reference": true, // Sora 特有字段
	}
	return knownFields[field]
}

// shouldConvertToMultipart 检查是否需要将 JSON 请求转换为 multipart/form-data
func shouldConvertToMultipart(model string, contentType string) bool {
	if !strings.HasPrefix(contentType, "application/json") {
		return false
	}

	// 检查 model 是否包含 sora 或 veo
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "sora") || strings.Contains(modelLower, "veo")
}

// convertJSONToMultipart 将 JSON 请求转换为 multipart/form-data
func convertJSONToMultipart(c *gin.Context) error {
	// 读取原始 JSON 请求体
	body, err := common.GetRequestBody(c)
	if err != nil {
		return fmt.Errorf("failed to get request body: %w", err)
	}
	return convertJSONToMultipartWithBody(c, body)
}

// convertJSONToMultipartWithBody 使用已经读取的 body 将 JSON 请求转换为 multipart/form-data
func convertJSONToMultipartWithBody(c *gin.Context, body []byte) error {
	// 解析 JSON
	var jsonReq map[string]interface{}
	if err := json.Unmarshal(body, &jsonReq); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// 创建一个 buffer 来存储 multipart 数据
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 转换字段并写入 multipart
	for key, value := range jsonReq {
		if key == "input" {
			// 特殊处理 input 对象 - 展开其中的字段
			if inputMap, ok := value.(map[string]interface{}); ok {
				for inputKey, inputValue := range inputMap {
					if inputKey == "input_reference" {
						// 处理 input_reference 字段 - 将数组中的每个URL作为独立字段写入
						switch v := inputValue.(type) {
						case string:
							// 单个字符串直接写入
							if v != "" {
								writer.WriteField(inputKey, v)
							}
						case []interface{}:
							// 处理接口数组，将每个URL作为独立字段
							for _, item := range v {
								if url, ok := item.(string); ok && url != "" {
									writer.WriteField(inputKey, url)
								}
							}
						case []string:
							// 处理字符串数组，将每个URL作为独立字段
							for _, url := range v {
								if url != "" {
									writer.WriteField(inputKey, url)
								}
							}
						default:
							// 其他类型，尝试序列化为JSON字符串
							if jsonBytes, err := json.Marshal(inputValue); err == nil {
								writer.WriteField(inputKey, string(jsonBytes))
							}
						}
					} else {
						// 处理 input 对象中的其他字段（如 prompt）
						if valueStr, ok := inputValue.(string); ok && valueStr != "" {
							writer.WriteField(inputKey, valueStr)
						} else if valueNum, ok := inputValue.(float64); ok {
							writer.WriteField(inputKey, strconv.FormatFloat(valueNum, 'f', -1, 64))
						} else if valueBool, ok := inputValue.(bool); ok {
							writer.WriteField(inputKey, strconv.FormatBool(valueBool))
						} else {
							// 其他类型序列化为JSON
							if jsonBytes, err := json.Marshal(inputValue); err == nil {
								writer.WriteField(inputKey, string(jsonBytes))
							}
						}
					}
				}
			}
		} else if key == "input_reference" {
			// 处理顶层的 input_reference 字段（兼容性）
			switch v := value.(type) {
			case string:
				if v != "" {
					writer.WriteField(key, v)
				}
			case []interface{}:
				for _, item := range v {
					if url, ok := item.(string); ok && url != "" {
						writer.WriteField(key, url)
					}
				}
			case []string:
				for _, url := range v {
					if url != "" {
						writer.WriteField(key, url)
					}
				}
			default:
				if jsonBytes, err := json.Marshal(value); err == nil {
					writer.WriteField(key, string(jsonBytes))
				}
			}
		} else if valueStr, ok := value.(string); ok {
			if valueStr != "" {
				writer.WriteField(key, valueStr)
			}
		} else if valueSlice, ok := value.([]interface{}); ok {
			// 处理数组类型，如 images
			for _, v := range valueSlice {
				if s, ok := v.(string); ok && s != "" {
					writer.WriteField(key, s)
				}
			}
		} else if valueNum, ok := value.(float64); ok {
			// 处理数字类型，如 duration
			writer.WriteField(key, strconv.FormatFloat(valueNum, 'f', -1, 64))
		} else if valueMap, ok := value.(map[string]interface{}); ok {
			// 处理 metadata 等复杂对象，将其序列化为 JSON 字符串
			if jsonBytes, err := json.Marshal(valueMap); err == nil {
				writer.WriteField(key, string(jsonBytes))
			}
		}
	}

	// 关闭 writer 以完成 multipart 写入
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 更新请求体
	multipartBody := buf.Bytes()
	c.Request.Body = io.NopCloser(bytes.NewReader(multipartBody))

	// 更新 Content-Type 为 multipart/form-data，包含 boundary
	contentType := writer.FormDataContentType()
	c.Request.Header.Set("Content-Type", contentType)

	// 重要：更新缓存，确保后续 GetRequestBody 调用返回正确的 multipart 数据
	c.Set(common.KeyRequestBody, multipartBody)

	// 调试信息
	fmt.Printf("DEBUG: Converted to multipart, Content-Type: %s, Body length: %d\n", contentType, len(multipartBody))
	fmt.Printf("DEBUG: Multipart body preview: %s\n", string(multipartBody[:min(200, len(multipartBody))]))

	// 注意：不要在这里调用 ParseMultipartForm，让后续的 c.ShouldBind 来处理
	return nil
}

func ValidateBasicTaskRequest(c *gin.Context, info *RelayInfo, action string) *dto.TaskError {
	var err error
	var req TaskSubmitReq

	// 先尝试获取 model 参数来判断是否需要转换
	contentType := c.GetHeader("Content-Type")

	// 如果是 JSON 请求，先尝试解析获取 model
	if strings.HasPrefix(contentType, "application/json") {
		var tempReq struct {
			Model string `json:"model"`
		}
		body, err := common.GetRequestBody(c)
		if err == nil {
			json.Unmarshal(body, &tempReq)

			// 检查是否需要转换为 multipart
			if shouldConvertToMultipart(tempReq.Model, contentType) {
				// 直接使用已经获取的 body 进行转换，避免重复读取
				if err := convertJSONToMultipartWithBody(c, body); err != nil {
					return createTaskError(err, "conversion_failed", http.StatusBadRequest, true)
				}
				// 转换后重新获取 Content-Type
				contentType = c.GetHeader("Content-Type")
			}
		}
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		req, err = validateMultipartTaskRequest(c, info, action)
		if err != nil {
			return createTaskError(err, "invalid_multipart_form", http.StatusBadRequest, true)
		}
	} else if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return createTaskError(err, "invalid_request", http.StatusBadRequest, true)
	}

	if taskErr := validatePrompt(req.Prompt); taskErr != nil {
		return taskErr
	}

	if len(req.Images) == 0 && strings.TrimSpace(req.Image) != "" {
		// 兼容单图上传
		req.Images = []string{req.Image}
	}

	storeTaskRequest(c, info, action, req)
	return nil
}
