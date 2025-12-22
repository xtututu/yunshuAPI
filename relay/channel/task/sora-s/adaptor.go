package soras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"xunkecloudAPI/dto"
	"xunkecloudAPI/relay/channel"
	relaycommon "xunkecloudAPI/relay/common"
)

// TaskAdaptor 实现任务适配器接口
type TaskAdaptor struct {
	apiKey      string
	channelId   int
	ChannelType int
	baseURL     string
}

// TaskRequest 请求参数结构体
type TaskRequest struct {
	Prompt      string `json:"prompt"`
	URL         string `json:"url"`
	AspectRatio string `json:"aspectRatio"`
	Duration    int    `json:"duration"`
	Size        string `json:"size"`
}

// TaskResponse 响应参数结构体
type TaskResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data TaskData `json:"data"`
}

// TaskData 响应数据结构体
type TaskData struct {
	ID string `json:"id"`
}

// TaskDetailResponse 任务详情响应结构体
type TaskDetailResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data TaskDetailData `json:"data"`
}

// TaskDetailData 任务详情数据结构体
type TaskDetailData struct {
	Content       string `json:"content"`
	Status        int    `json:"status"`
	FailReason    string `json:"fail_reason"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	RemoteURL     string `json:"remote_url"`
	Size          string `json:"size"`
	Duration      int    `json:"duration"`
	AspectRatio   string `json:"aspectRatio"`
	URL           string `json:"url"`
	PID           string `json:"pid"`
	RemixTargetId string `json:"remixTargetId"`
	TransferURL   string `json:"transfer_url"`
	ID            string `json:"id"`
}

// Init 初始化任务适配器
func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	// 使用配置的ChannelBaseUrl，如果为空则使用默认值
	if info.ChannelBaseUrl != "" {
		a.baseURL = info.ChannelBaseUrl
	} else {
		a.baseURL = "https://api.wuyinkeji.com"
	}
	a.apiKey = info.ApiKey
	a.channelId = info.ChannelId
}

// NewTaskAdaptor 创建新的任务适配器
func NewTaskAdaptor(info *relaycommon.RelayInfo) *TaskAdaptor {
	a := &TaskAdaptor{}
	a.Init(info)
	return a
}

// ValidateRequestAndSetAction 验证请求并设置操作
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateMultipartDirect(c, info)
}

// BuildRequestURL 构建请求URL
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 根据模型选择不同的API端点
	switch info.OriginModelName {
	case ModelSora2Pro:
		return APIEndpointSora2Pro, nil
	case ModelSora2, ModelSora2HD:
		return APIEndpointSora2, nil
	default:
		return APIEndpointSora2, nil
	}
}

// BuildRequestHeader 构建请求头
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	// 设置API密钥
	req.Header.Set("X-API-KEY", a.apiKey)
	// 设置Authorization头，不加Bearer前缀直接传递API密钥
	req.Header.Set("Authorization", info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// BuildRequestBody 构建请求体
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	// 获取任务请求参数
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, fmt.Errorf("get_task_request_failed: %w", err)
	}

	// 构建请求参数
	requestPayload := TaskRequest{
		Prompt:      req.Prompt,
		URL:         req.URL,
		AspectRatio: req.AspectRatio,
		Size:        req.Size,
	}

	// 处理URL优先级
	if requestPayload.URL == "" && req.InputReference != nil {
		switch v := req.InputReference.(type) {
		case string:
			if v != "" {
				requestPayload.URL = v
			}
		case []string:
			if len(v) > 0 {
				requestPayload.URL = v[0]
			}
		case []interface{}:
			if len(v) > 0 {
				if str, ok := v[0].(string); ok && str != "" {
					requestPayload.URL = str
				}
			}
		}
	}

	// 处理duration
	duration := 10 // 默认10秒
	if req.Duration != 0 {
		duration = req.Duration
	} else if req.Seconds != "" {
		if d, err := strconv.Atoi(req.Seconds); err == nil {
			duration = d
		}
	}
	requestPayload.Duration = duration

	// 处理aspectRatio
	if requestPayload.AspectRatio == "" {
		// 根据size推断aspectRatio
		switch requestPayload.Size {
		case "small":
			fallthrough
		case "medium":
			fallthrough
		case "large":
			requestPayload.AspectRatio = "16:9" // 默认16:9
		}
	}

	// 转换为JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal_request_payload_failed: %w", err)
	}

	return bytes.NewBuffer(jsonData), nil
}

// ParseResponse 解析响应（兼容旧接口）
func (a *TaskAdaptor) ParseResponse(resp *http.Response) (string, *relaycommon.TaskInfo, error) {
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read_response_body_failed: %w", err)
	}

	// 解析响应
	var response TaskResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", nil, fmt.Errorf("unmarshal_response_failed: %w", err)
	}

	// 检查响应状态
	if response.Code != 200 {
		return "", nil, fmt.Errorf("api_error: %s", response.Msg)
	}

	// 构建任务信息
	taskInfo := &relaycommon.TaskInfo{
		TaskID:   response.Data.ID,
		Status:   "PENDING",
		Progress: "0%",
	}

	return response.Data.ID, taskInfo, nil
}

// GetTaskStatus 获取任务状态
func (a *TaskAdaptor) GetTaskStatus(taskID string, info *relaycommon.RelayInfo) (taskInfo *relaycommon.TaskInfo, err error) {
	// 构建请求URL
	url := fmt.Sprintf("%s?id=%s", APIEndpointDetail, taskID)

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create_request_failed: %w", err)
	}

	// 设置请求头
	req.Header.Set("X-API-KEY", a.apiKey)
	// 设置Authorization头，不加Bearer前缀直接传递API密钥
	req.Header.Set("Authorization", a.apiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send_request_failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read_response_body_failed: %w", err)
	}

	// 解析响应
	var response TaskDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal_response_failed: %w", err)
	}

	// 检查响应状态
	if response.Code != 200 {
		return nil, fmt.Errorf("api_error: %s", response.Msg)
	}

	// 构建任务信息
	taskInfo = &relaycommon.TaskInfo{
		TaskID:    response.Data.ID,
		Status:    convertStatus(response.Data.Status),
		Progress:  convertProgress(response.Data.Status),
		RemoteUrl: response.Data.RemoteURL,
		Url:       response.Data.URL,
	}

	// 如果有transfer_url，优先使用它作为RemoteUrl
	if response.Data.TransferURL != "" {
		taskInfo.RemoteUrl = response.Data.TransferURL
	}

	// 如果任务失败，添加失败原因
	if response.Data.Status == 2 { // API返回的失败状态码
		taskInfo.Reason = response.Data.FailReason
	}

	return taskInfo, nil
}

// BuildTaskInfo 构建任务信息
func (a *TaskAdaptor) BuildTaskInfo(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*relaycommon.TaskInfo, error) {
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	// 解析响应
	var response TaskResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	// 检查响应状态
	if response.Code != 200 {
		return nil, fmt.Errorf("API error: %s", response.Msg)
	}

	// 构建任务信息
	return &relaycommon.TaskInfo{
		TaskID:   response.Data.ID,
		Status:   "PENDING",
		Progress: "0%",
	}, nil
}

// DoRequest 执行请求
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse 处理响应
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = &dto.TaskError{Code: "500", Message: "read_response_body_failed: " + err.Error()}
		return
	}
	_ = resp.Body.Close()

	// 解析响应
	var response TaskResponse
	if err := json.Unmarshal(body, &response); err != nil {
		taskErr = &dto.TaskError{Code: "500", Message: "unmarshal_response_failed: " + err.Error()}
		return
	}

	// 检查响应状态
	if response.Code != 200 {
		taskErr = &dto.TaskError{Code: fmt.Sprintf("%d", response.Code), Message: response.Msg}
		return
	}

	// 返回任务ID和响应数据
	return response.Data.ID, body, nil
}

// GetModelList 获取支持的模型列表
func (a *TaskAdaptor) GetModelList() []string {
	return []string{ModelSora2, ModelSora2HD, ModelSora2Pro}
}

// GetChannelName 获取渠道名称
func (a *TaskAdaptor) GetChannelName() string {
	return "sora-s"
}

// FetchTask 获取任务状态
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	// 从body中获取task_id
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	// 构建请求URL - 优先使用传入的baseUrl参数
	useBaseUrl := baseUrl
	if useBaseUrl == "" {
		useBaseUrl = a.baseURL
	}
	url := fmt.Sprintf("%s/api/sora2/detail?id=%s", useBaseUrl, taskID)

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create_request_failed: %w", err)
	}

	// 设置请求头 - 优先使用传入的key参数
	useApiKey := key
	if useApiKey == "" {
		useApiKey = a.apiKey
	}
	req.Header.Set("X-API-KEY", useApiKey)
	// 设置Authorization头，不加Bearer前缀直接传递API密钥
	req.Header.Set("Authorization", useApiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send_request_failed: %w", err)
	}

	return resp, nil
}

// ParseTaskResult 解析任务结果
func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	// 解析响应
	var response TaskDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal_response_failed: %w", err)
	}

	// 检查响应状态
	if response.Code != 200 {
		return nil, fmt.Errorf("api_error: %s", response.Msg)
	}

	// 转换任务状态
	status := "PENDING"
	progress := "0%"
	switch response.Data.Status {
	case 0: // API返回的排队中
		status = "PENDING"
		progress = "0%"
	case 1: // API返回的成功
		status = "SUCCESS"
		progress = "100%"
	case 2: // API返回的失败
		status = "FAILURE"
		progress = "100%"
	case 3: // API返回的生成中
		status = "PROCESSING"
		progress = "50%"
	default:
		status = "UNKNOWN"
		progress = "0%"
	}

	// 构建任务信息
	taskInfo := &relaycommon.TaskInfo{
		TaskID:    response.Data.ID,
		Status:    status,
		Progress:  progress,
		RemoteUrl: response.Data.RemoteURL,
		Url:       response.Data.URL,
	}

	// 如果任务失败，添加失败原因
	if status == "FAILURE" {
		taskInfo.Reason = response.Data.FailReason
	}

	return taskInfo, nil
}

// 辅助函数：转换状态码
func convertStatus(status int) string {
	switch status {
	case 0: // API返回的排队中
		return "PENDING"
	case 1: // API返回的成功
		return "SUCCESS"
	case 2: // API返回的失败
		return "FAILURE"
	case 3: // API返回的生成中
		return "PROCESSING"
	default:
		return "UNKNOWN"
	}
}

// 辅助函数：转换进度
func convertProgress(status int) string {
	switch status {
	case 0: // API返回的排队中
		return "0%"
	case 1: // API返回的成功
		return "100%"
	case 2: // API返回的失败
		return "100%"
	case 3: // API返回的生成中
		return "50%"
	default:
		return "0%"
	}
}
