package sorag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"xunkecloudAPI/common"
	common2 "xunkecloudAPI/common"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/model"
	"xunkecloudAPI/relay/channel"
	relaycommon "xunkecloudAPI/relay/common"
	"xunkecloudAPI/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================// Request / Response structures// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

// API请求结构
type createVideoRequest struct {
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	URL           string   `json:"url"`
	AspectRatio   string   `json:"aspectRatio"`
	Duration      int      `json:"duration"`
	RemixTargetId string   `json:"remixTargetId"`
	Characters    []string `json:"characters"`
	Size          string   `json:"size"`
	WebHook       string   `json:"webHook"`
	ShutProgress  bool     `json:"shutProgress"`
}

// 创建任务响应结构
type createResponse struct {
	Code int `json:"code"`
	Data struct {
		ID      string `json:"id"`
		Results []struct {
			URL             string `json:"url"`
			RemoveWatermark bool   `json:"removeWatermark"`
			PID             string `json:"pid"`
		} `json:"results"`
		Progress      int    `json:"progress"`
		Status        string `json:"status"`
		FailureReason string `json:"failure_reason"`
		Error         string `json:"error"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// 轮询请求结构
type pollRequest struct {
	ID string `json:"id"`
}

// 轮询响应结构
type pollResponse struct {
	Code int `json:"code"`
	Data struct {
		ID      string `json:"id"`
		Results []struct {
			URL             string `json:"url"`
			RemoveWatermark bool   `json:"removeWatermark"`
			PID             string `json:"pid"`
		} `json:"results"`
		Progress      int    `json:"progress"`
		Status        string `json:"status"`
		FailureReason string `json:"failure_reason"`
		Error         string `json:"error"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// ============================// Adaptor implementation// ============================

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	// 直接设置baseURL，不依赖其他值
	a.baseURL = "https://grsai.dakka.com.cn"
	if common2.DebugEnabled {
		println("Sora-g Adaptor Init - hardcoded baseURL:", a.baseURL)
	}
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// 确保baseURL不为空
	if a.baseURL == "" {
		errorMsg := "Sora-g BuildRequestURL - baseURL is empty"
		// 在非调试模式下也输出错误信息，方便Docker环境调试
		println(errorMsg)
		return "", fmt.Errorf(errorMsg)
	}

	url := fmt.Sprintf("%s/v1/video/sora-video", a.baseURL)

	// 在非调试模式下也输出URL信息，方便Docker环境调试
	println("Sora-g BuildRequestURL - baseURL:", a.baseURL)
	println("Sora-g BuildRequestURL - full URL:", url)

	return url, nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	// 获取任务请求参数
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_task_request_failed")
	}

	// 构建sora-g请求体
	requestPayload := struct {
		Model         string `json:"model"`
		Prompt        string `json:"prompt"`
		URL           string `json:"url"`
		AspectRatio   string `json:"aspectRatio"`
		Duration      int    `json:"duration"`
		RemixTargetId string `json:"remixTargetId"`
		Characters    []any  `json:"characters"`
		Size          string `json:"size"`
		WebHook       string `json:"webHook"`
		ShutProgress  bool   `json:"shutProgress"`
	}{
		Model:         req.Model,
		Prompt:        req.Prompt,
		URL:           req.URL, // 优先使用直接传入的URL
		AspectRatio:   req.AspectRatio,
		Duration:      req.Duration,
		RemixTargetId: "",
		Characters:    []any{},
		Size:          "large", // 固定设置为large
		WebHook:       "-1",    // 固定设置为-1
		ShutProgress:  true,    // 固定设置为true
	}

	// 映射模型名
	if info.IsModelMapped && info.UpstreamModelName != "" {
		requestPayload.Model = info.UpstreamModelName
	}

	// 如果URL为空，尝试从input_reference获取
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

	// 映射seconds到duration
	if req.Seconds != "" {
		if duration, err := strconv.Atoi(req.Seconds); err == nil {
			requestPayload.Duration = duration
		}
	}

	// 如果seconds没有设置但duration设置了，使用duration
	if requestPayload.Duration == 0 && req.Duration > 0 {
		requestPayload.Duration = req.Duration
	}

	// 映射size到aspectRatio
	if req.Size != "" {
		requestPayload.AspectRatio = a.getAspectRatio(req.Size)
	}

	// 将请求体序列化为JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal_request_payload_failed")
	}

	// 创建一个新的 bytes.Buffer 来支持多次读取
	bodyCopy := bytes.NewBuffer(requestBody)
	return bodyCopy, nil
}

// DoRequest delegates to common helper.
// getAspectRatio converts size to aspectRatio
func (a *TaskAdaptor) getAspectRatio(size string) string {
	switch size {
	case "16:9":
		return "16:9"
	case "9:16":
		return "9:16"
	case "1:1":
		return "1:1"
	case "4:3":
		return "4:3"
	case "3:4":
		return "3:4"
	case "small":
		return "16:9"
	case "medium":
		return "16:9"
	case "large":
		return "16:9"
	default:
		return "16:9" // 默认值
	}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse response
	var createResp struct {
		Code int `json:"code"`
		Data struct {
			ID      string `json:"id"`
			Results []struct {
				URL             string `json:"url"`
				RemoveWatermark bool   `json:"removeWatermark"`
				Pid             string `json:"pid"`
			} `json:"results"`
			Progress      int    `json:"progress"`
			Status        string `json:"status"`
			FailureReason string `json:"failure_reason"`
			Error         string `json:"error"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := common.Unmarshal(responseBody, &createResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if createResp.Code != 0 {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("API error: %s", createResp.Msg), "api_error", http.StatusInternalServerError)
		return
	}

	if createResp.Data.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	return createResp.Data.ID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/draw/result", baseUrl)

	// 构建轮询请求体
	pollReq := struct {
		ID string `json:"id"`
	}{
		ID: taskID,
	}

	reqBody, err := json.Marshal(pollReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// 解析轮询响应
	var pollResp pollResponse
	if err := common.Unmarshal(respBody, &pollResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	if pollResp.Code != 0 {
		taskResult.Status = model.TaskStatusFailure
		taskResult.Reason = pollResp.Msg
		return &taskResult, nil
	}

	// 检查任务状态
	switch pollResp.Data.Status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "succeeded", "completed":
		taskResult.Status = model.TaskStatusSuccess
		if len(pollResp.Data.Results) > 0 {
			taskResult.Url = pollResp.Data.Results[0].URL
		}
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if pollResp.Data.FailureReason != "" {
			taskResult.Reason = pollResp.Data.FailureReason
		} else if pollResp.Data.Error != "" {
			taskResult.Reason = pollResp.Data.Error
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		// 未知状态默认为处理中
		taskResult.Status = model.TaskStatusInProgress
	}

	// 设置进度
	if pollResp.Data.Progress > 0 && pollResp.Data.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", pollResp.Data.Progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var soraResp struct {
		Data struct {
			VideoURL string `json:"video_url"`
		}
	}
	if err := json.Unmarshal(task.Data, &soraResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal sora-g task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	openAIVideo.CreatedAt = task.CreatedAt
	openAIVideo.CompletedAt = task.UpdatedAt

	if task.Status == model.TaskStatusSuccess {
		openAIVideo.SetMetadata("url", soraResp.Data.VideoURL)
	}

	if task.Status == model.TaskStatusFailure {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: task.FailReason,
			Code:    "task_failed",
		}
	}

	return common.Marshal(openAIVideo)
}
