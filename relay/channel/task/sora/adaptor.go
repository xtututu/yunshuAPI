package sora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"yunshuAPI/common"
	"yunshuAPI/dto"
	"yunshuAPI/model"
	"yunshuAPI/relay/channel"
	relaycommon "yunshuAPI/relay/common"
	"yunshuAPI/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string `json:"id"`
	TaskID             string `json:"task_id,omitempty"` //兼容旧接口
	Object             string `json:"object"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	CreatedAt          int64  `json:"created_at"`
	CompletedAt        int64  `json:"completed_at,omitempty"`
	ExpiresAt          int64  `json:"expires_at,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	VideoURL           string `json:"video_url,omitempty"` // 视频URL
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type grokResponseTask struct {
	ID               string `json:"id"`
	Mode             string `json:"mode"`
	Type             string `json:"type"`
	Error            string `json:"error"`
	Model            string `json:"model"`
	Ratio            string `json:"ratio"`
	Prompt           string `json:"prompt"`
	Status           string `json:"status"`
	PostID           string `json:"post_id"`
	AssetID          string `json:"asset_id"`
	Progress         int    `json:"progress"`
	TraceID          string `json:"trace_id"`
	Upscaled         bool   `json:"upscaled"`
	VideoID          string `json:"video_id"`
	VideoURL         string `json:"video_url"`
	CompletedAt      int64  `json:"completed_at"`
	ThumbnailURL     string `json:"thumbnail_url"`
	VideoFileID      string `json:"video_file_id"`
	ThumbnailFileID  string `json:"thumbnail_file_id"`
	StatusUpdateTime int64  `json:"status_update_time"`
	UpscaleOnComplete bool  `json:"upscale_on_complete"`
}

// ============================// Adaptor implementation// ============================

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
	newBoundary string
}

func isGrokModel(model string) bool {
	return model == "grok-video-3" || model == "grok-video-3-10s"
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelName := info.UpstreamModelName
	if isGrokModel(modelName) {
		return fmt.Sprintf("%s/v1/video/create", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	// 获取原始Content-Type
	contentType := c.Request.Header.Get("Content-Type")

	// 如果重新构建了multipart请求体，使用新的boundary更新Content-Type
	if strings.HasPrefix(contentType, "multipart/form-data") && a.newBoundary != "" {
		// 解析原始Content-Type
		mediatype, params, err := mime.ParseMediaType(contentType)
		if err == nil {
			// 更新boundary
			params["boundary"] = a.newBoundary
			// 重新构建Content-Type
			contentType = mime.FormatMediaType(mediatype, params)
			// 重置newBoundary，避免影响后续请求
			a.newBoundary = ""
		}
	}

	req.Header.Set("Content-Type", contentType)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	modelName := info.UpstreamModelName
	
	cachedBody, err := common.GetRequestBody(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}

	contentType := c.GetHeader("Content-Type")

	if isGrokModel(modelName) {
		var jsonReq map[string]interface{}
		if err := json.Unmarshal(cachedBody, &jsonReq); err != nil {
			return nil, errors.Wrap(err, "parse_json_request_failed")
		}

		grokReq := make(map[string]interface{})
		
		grokReq["model"] = modelName
		
		if prompt, ok := jsonReq["prompt"].(string); ok {
			grokReq["prompt"] = prompt 
		}
		
		if size, ok := jsonReq["size"].(string); ok {
			grokReq["aspect_ratio"] = size
		}
		
		grokReq["size"] = "720P"
		
		if inputRef, hasInputRef := jsonReq["input_reference"]; hasInputRef {
			var images []string
			switch v := inputRef.(type) {
			case string:
				cleanURL := strings.Trim(v, " `")
				if cleanURL != "" {
					images = append(images, cleanURL)
				}
			case []interface{}:
				for _, item := range v {
					if url, ok := item.(string); ok {
						cleanURL := strings.Trim(url, " `")
						if cleanURL != "" {
							images = append(images, cleanURL)
						}
					}
				}
			}
			if len(images) > 0 {
				grokReq["images"] = images
			}
		}

		updatedBody, err := json.Marshal(grokReq)
		if err != nil {
			return nil, errors.Wrap(err, "marshal_grok_request_failed")
		}

		bodyCopy := bytes.NewBuffer(updatedBody)
		return bodyCopy, nil
	}

	if strings.HasPrefix(contentType, "application/json") {
		var jsonReq map[string]interface{}
		if err := json.Unmarshal(cachedBody, &jsonReq); err != nil {
			return nil, errors.Wrap(err, "parse_json_request_failed")
		}

		inputReference, hasInputRef := jsonReq["input_reference"]
		if hasInputRef {
			var newBody bytes.Buffer
			writer := multipart.NewWriter(&newBody)

			a.newBoundary = writer.Boundary()

			var urls []string

			switch v := inputReference.(type) {
			case string:
				urls = []string{v}
			case []interface{}:
				for _, item := range v {
					if url, ok := item.(string); ok {
						urls = append(urls, url)
					}
				}
			}

			for i, url := range urls {
				cleanURL := strings.Trim(url, " `")
				if cleanURL == "" {
					continue
				}

				resp, err := service.DoDownloadRequest(cleanURL)
				if err != nil {
					writer.Close()
					return nil, errors.Wrapf(err, "download_image_failed: %s", cleanURL)
				}
				defer resp.Body.Close()

				imageData, err := io.ReadAll(resp.Body)
				if err != nil {
					writer.Close()
					return nil, errors.Wrapf(err, "read_image_failed: %s", cleanURL)
				}

				filename := fmt.Sprintf("image_%d.jpg", i+1)

				part, err := writer.CreateFormFile("input_reference", filename)
				if err != nil {
					writer.Close()
					return nil, errors.Wrapf(err, "create_form_file_failed: %s", filename)
				}

				if _, err := part.Write(imageData); err != nil {
					writer.Close()
					return nil, errors.Wrapf(err, "write_image_failed: %s", filename)
				}
			}

			delete(jsonReq, "input_reference")

			for key, value := range jsonReq {
				var strValue string
				switch v := value.(type) {
				case string:
					strValue = v
				case float64:
					strValue = fmt.Sprintf("%.0f", v)
				case bool:
					strValue = fmt.Sprintf("%t", v)
				default:
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						writer.Close()
						return nil, errors.Wrapf(err, "marshal_value_failed: %v", v)
					}
					strValue = string(jsonBytes)
				}

				writer.WriteField(key, strValue)
			}

			writer.Close()

			bodyCopy := bytes.NewBuffer(newBody.Bytes())
			return bodyCopy, nil
		}
	}

	if info.IsModelMapped && info.UpstreamModelName != "" {
		if strings.HasPrefix(contentType, "multipart/form-data") {
			_, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				return nil, errors.Wrap(err, "parse_content_type_failed")
			}
			boundary, ok := params["boundary"]
			if !ok {
				return nil, errors.New("boundary_not_found_in_content_type")
			}
			reader := multipart.NewReader(bytes.NewReader(cachedBody), boundary)
			parts, err := reader.ReadForm(10 << 20)
			if err != nil {
				return nil, errors.Wrap(err, "parse_multipart_form_failed")
			}

			var newBody bytes.Buffer
			writer := multipart.NewWriter(&newBody)

			a.newBoundary = writer.Boundary()

			for key, values := range parts.Value {
				for _, value := range values {
					if key == "model" {
						writer.WriteField(key, info.UpstreamModelName)
					} else {
						writer.WriteField(key, value)
					}
				}
			}

			for key, files := range parts.File {
				for _, file := range files {
					fileContent, err := file.Open()
					if err != nil {
						return nil, errors.Wrap(err, "open_file_failed")
					}

					fileHeader := make([]byte, file.Size)
					_, err = fileContent.Read(fileHeader)
					if err != nil {
						fileContent.Close()
						return nil, errors.Wrap(err, "read_file_failed")
					}

					fileContent.Close()

					part, err := writer.CreateFormFile(key, file.Filename)
					if err != nil {
						return nil, errors.Wrap(err, "create_form_file_failed")
					}

					part.Write(fileHeader)
				}
			}

			writer.Close()

			bodyCopy := bytes.NewBuffer(newBody.Bytes())
			return bodyCopy, nil
		} else if strings.HasPrefix(contentType, "application/json") {
			var jsonReq map[string]interface{}
			if err := json.Unmarshal(cachedBody, &jsonReq); err != nil {
				return nil, errors.Wrap(err, "parse_json_request_failed")
			}

			jsonReq["model"] = info.UpstreamModelName

			updatedBody, err := json.Marshal(jsonReq)
			if err != nil {
				return nil, errors.Wrap(err, "marshal_json_request_failed")
			}

			bodyCopy := bytes.NewBuffer(updatedBody)
			return bodyCopy, nil
		}
	}

	bodyCopy := bytes.NewBuffer(cachedBody)
	return bodyCopy, nil
}

// DoRequest delegates to common helper.
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

	var grokResp grokResponseTask
	if err := common.Unmarshal(responseBody, &grokResp); err == nil && grokResp.ID != "" {
		if grokResp.VideoID != "" {
			return grokResp.VideoID, responseBody, nil
		}
		return grokResp.ID, responseBody, nil
	}

	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		if dResp.TaskID == "" {
			taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
			return
		}
		dResp.ID = dResp.TaskID
		dResp.TaskID = ""
	}

	return dResp.ID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	var grokResp grokResponseTask
	if err := common.Unmarshal(respBody, &grokResp); err == nil && grokResp.ID != "" {
		switch grokResp.Status {
		case "queued", "pending":
			taskResult.Status = model.TaskStatusQueued
		case "processing", "in_progress":
			taskResult.Status = model.TaskStatusInProgress
		case "completed":
			taskResult.Status = model.TaskStatusSuccess
			if grokResp.VideoURL != "" {
				taskResult.Url = grokResp.VideoURL
				taskResult.Reason = grokResp.VideoURL
			}
		case "failed", "cancelled":
			taskResult.Status = model.TaskStatusFailure
			if grokResp.Error != "" {
				taskResult.Reason = grokResp.Error
			} else {
				taskResult.Reason = "task failed"
			}
		default:
		}
		if grokResp.Progress > 0 && grokResp.Progress < 100 {
			taskResult.Progress = fmt.Sprintf("%d%%", grokResp.Progress)
		}
		return &taskResult, nil
	}

	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	switch resTask.Status {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		if resTask.VideoURL != "" {
			taskResult.Url = resTask.VideoURL
			taskResult.Reason = resTask.VideoURL
		} else {
			taskResult.Url = fmt.Sprintf("%s/v1/videos/%s/content", a.baseURL, resTask.ID)
			taskResult.Reason = taskResult.Url
		}
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}

	return &taskResult, nil
}

// 定义用户期望的视频响应结构体
type UserExpectedVideoResponse struct {
	ID         string `json:"id"`
	Size       string `json:"size"`
	Model      string `json:"model"`
	Object     string `json:"object"`
	Status     string `json:"status"`
	Seconds    string `json:"seconds"`
	Progress   int    `json:"progress"`
	VideoURL   string `json:"video_url"`
	CreatedAt  int64  `json:"created_at"`
	FailReason string `json:"fail_reason,omitempty"`
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	response := UserExpectedVideoResponse{
		ID:        task.TaskID,
		Object:    "video",
		Status:    task.Status.ToVideoStatus(),
		Progress:  100,
		CreatedAt: task.CreatedAt,
	}

	var grokResp grokResponseTask
	if err := json.Unmarshal(task.Data, &grokResp); err == nil && grokResp.ID != "" {
		response.Model = grokResp.Model
		response.Seconds = ""
		response.Size = grokResp.Ratio

		if task.Status == model.TaskStatusSuccess {
			if grokResp.VideoURL != "" {
				response.VideoURL = grokResp.VideoURL
			}
		} else if task.Status == model.TaskStatusFailure {
			if grokResp.Error != "" {
				response.FailReason = grokResp.Error
			} else if task.FailReason != "" {
				response.FailReason = task.FailReason
			} else {
				response.FailReason = "Unknown error"
			}
		}
		return common.Marshal(response)
	}

	var soraResp responseTask
	if err := json.Unmarshal(task.Data, &soraResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal sora task data failed")
	}

	response.Size = soraResp.Size
	response.Model = soraResp.Model
	response.Seconds = soraResp.Seconds

	if task.Status == model.TaskStatusSuccess {
		if soraResp.VideoURL != "" {
			response.VideoURL = soraResp.VideoURL
		} else {
			response.VideoURL = fmt.Sprintf("%s/v1/videos/%s/content", a.baseURL, task.TaskID)
		}
	} else if task.Status == model.TaskStatusFailure {
		if soraResp.Error != nil && soraResp.Error.Message != "" {
			response.FailReason = soraResp.Error.Message
		} else if task.FailReason != "" {
			response.FailReason = task.FailReason
		} else {
			response.FailReason = "Unknown error"
		}
	}

	return common.Marshal(response)
}
