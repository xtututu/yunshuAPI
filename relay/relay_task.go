package relay

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
	"yunshuAPI/constant"
	"yunshuAPI/dto"
	"yunshuAPI/logger"
	"yunshuAPI/model"
	"yunshuAPI/relay/channel"
	relaycommon "yunshuAPI/relay/common"
	relayconstant "yunshuAPI/relay/constant"
	"yunshuAPI/relay/helper"
	"yunshuAPI/service"
	"yunshuAPI/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

/*
Task 任务通过平台、Action 区分任务
*/
func RelayTaskSubmit(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	// ensure TaskRelayInfo is initialized to avoid nil dereference when accessing embedded fields
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	platform := constant.TaskPlatform(c.GetString("platform"))
	if platform == "" {
		platform = GetTaskPlatform(c)
	}

	info.InitChannelMeta(c)
	adaptor := GetTaskAdaptor(platform)
	// 检测是否是速创渠道
	isSuchuangChannel := false
	if adaptor == nil {
		// 特殊处理速创渠道，它实现的是channel.Adaptor接口而不是channel.TaskAdaptor接口
		channelType, err := strconv.Atoi(string(platform))
		if err != nil {
			return service.TaskErrorWrapperLocal(fmt.Errorf("invalid api platform: %s", platform), "invalid_api_platform", http.StatusBadRequest)
		}
		if channelType == constant.ChannelTypeSuchuang {
			// 对于速创渠道，直接使用GetAdaptor获取channel.Adaptor接口的实现
			// 因为速创渠道的图片生成请求在后面会被特殊处理（直接返回响应体）
			adaptor = nil
			isSuchuangChannel = true
			// 设置info.ChannelId，确保后面的逻辑能够正确识别
			info.ChannelId = constant.ChannelTypeSuchuang
		} else {
			return service.TaskErrorWrapperLocal(fmt.Errorf("invalid api platform: %s", platform), "invalid_api_platform", http.StatusBadRequest)
		}
	} else {
		adaptor.Init(info)
	}
	modelName := info.OriginModelName
	if modelName == "" {
		modelName = service.CoverTaskActionToModelName(platform, info.Action)
	}

	// Apply model mapping if available (before validation to ensure correct model is used)
	err := helper.ModelMappedHelper(c, info, nil)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "model_mapping_failed", http.StatusInternalServerError)
	}

	// get & validate taskRequest 获取并验证文本请求
	if adaptor != nil {
		taskErr = adaptor.ValidateRequestAndSetAction(c, info)
		if taskErr != nil {
			return
		}
	}

	// Use mapped model name for price calculation
	mappedModelName := info.UpstreamModelName
	if mappedModelName == "" {
		mappedModelName = modelName
	}

	modelPrice, success := ratio_setting.GetModelPrice(mappedModelName, true)
	if !success {
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[mappedModelName]
		if !ok {
			modelPrice = 0.1
		} else {
			modelPrice = defaultPrice
		}
	}

	// 预扣
	groupRatio := ratio_setting.GetGroupRatio(info.UsingGroup)
	var ratio float64
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)
	if hasUserGroupRatio {
		ratio = modelPrice * userGroupRatio
	} else {
		ratio = modelPrice * groupRatio
	}
	// FIXME: 临时修补，支持任务仅按次计费
	if !common.StringsContains(constant.TaskPricePatches, modelName) {
		if len(info.PriceData.OtherRatios) > 0 {
			for _, ra := range info.PriceData.OtherRatios {
				if 1.0 != ra {
					ratio *= ra
				}
			}
		}
	}
	println(fmt.Sprintf("model: %s, model_price: %.4f, group: %s, group_ratio: %.4f, final_ratio: %.4f", modelName, modelPrice, info.UsingGroup, groupRatio, ratio))
	userQuota, err := model.GetUserQuota(info.UserId, false)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
		return
	}
	quota := int(ratio * common.QuotaPerUnit)
	if userQuota-quota < 0 {
		taskErr = service.TaskErrorWrapperLocal(errors.New("user quota is not enough"), "quota_not_enough", http.StatusForbidden)
		return
	}

	if info.OriginTaskID != "" {
		originTask, exist, err := model.GetByTaskId(info.UserId, info.OriginTaskID)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "get_origin_task_failed", http.StatusInternalServerError)
			return
		}
		if !exist {
			taskErr = service.TaskErrorWrapperLocal(errors.New("task_origin_not_exist"), "task_not_exist", http.StatusBadRequest)
			return
		}
		if originTask.ChannelId != info.ChannelId {
			channel, err := model.GetChannelById(originTask.ChannelId, true)
			if err != nil {
				taskErr = service.TaskErrorWrapperLocal(err, "channel_not_found", http.StatusBadRequest)
				return
			}
			if channel.Status != common.ChannelStatusEnabled {
				return service.TaskErrorWrapperLocal(errors.New("该任务所属渠道已被禁用"), "task_channel_disable", http.StatusBadRequest)
			}
			c.Set("base_url", channel.GetBaseURL())
			c.Set("channel_id", originTask.ChannelId)
			// 根据渠道类型决定是否添加Bearer前缀
			authHeader := channel.Key
			if channel.Type != constant.ChannelTypeOpenAIS {
				authHeader = fmt.Sprintf("Bearer %s", channel.Key)
			}
			c.Request.Header.Set("Authorization", authHeader)

			info.ChannelBaseUrl = channel.GetBaseURL()
			info.ChannelId = originTask.ChannelId
		}
	}

	// 处理请求体和发送请求
	var resp *http.Response
	var requestBody io.Reader

	// 检查是否是速创渠道
	if isSuchuangChannel || info.ChannelId == constant.ChannelTypeSuchuang {
		// 对于速创渠道，使用channel.Adaptor接口的实现
		channelAdaptor := GetAdaptor(constant.APITypeSuchuang)
		channelAdaptor.Init(info)

		// 读取请求体内容，以便多次使用
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "read_request_body_failed", http.StatusBadRequest)
			return
		}

		// 解析原始请求为GeneralOpenAIRequest
		var generalRequest dto.GeneralOpenAIRequest
		if err := json.Unmarshal(bodyBytes, &generalRequest); err != nil {
			taskErr = service.TaskErrorWrapper(err, "unmarshal_request_failed", http.StatusBadRequest)
			return
		}

		// 转换OpenAI请求到速创API请求格式
		convertedRequest, err := channelAdaptor.ConvertOpenAIRequest(c, info, &generalRequest)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
			return
		}

		// 将转换后的请求序列化为JSON
		convertedBody, err := json.Marshal(convertedRequest)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "marshal_converted_request_failed", http.StatusInternalServerError)
			return
		}

		// 使用转换后的请求体调用DoRequest方法
		result, err := channelAdaptor.DoRequest(c, info, bytes.NewBuffer(convertedBody))
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
			return
		}

		// 检查result类型
		if httpResp, ok := result.(*http.Response); ok {
			// 如果是http.Response类型，调用DoResponse方法处理响应
			defer httpResp.Body.Close()

			// 先保存原始响应体
			originalBody, readErr := io.ReadAll(httpResp.Body)
			if readErr != nil {
				taskErr = service.TaskErrorWrapper(readErr, "read_original_response_body_failed", http.StatusInternalServerError)
				return
			}

			// 重置响应体，以便DoResponse方法可以读取
			httpResp.Body = io.NopCloser(bytes.NewBuffer(originalBody))
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Before DoResponse - Response body length: %d", len(originalBody))
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Before DoResponse - Content-Length: %d", httpResp.ContentLength)

			// 调用DoResponse方法处理响应
			_, doRespErr := channelAdaptor.DoResponse(c, httpResp, info)
			if doRespErr != nil {
				taskErr = service.TaskErrorWrapper(doRespErr, "do_response_failed", http.StatusInternalServerError)
				return
			}

			// 记录DoResponse后的响应头信�?
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] After DoResponse - Content-Length: %d", httpResp.ContentLength)
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] After DoResponse - Content-Type: %s", httpResp.Header.Get("Content-Type"))
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] After DoResponse - Headers: %v", httpResp.Header)

			// 重新读取转换后的响应�?
			body, readErr := io.ReadAll(httpResp.Body)
			if readErr != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("[SUCHUANG] Failed to read converted response body: %v", readErr))
				taskErr = service.TaskErrorWrapper(readErr, "read_converted_response_body_failed", http.StatusInternalServerError)
				return
			}

			// 调试日志：查看原始和转换后的响应�?
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Original response body: %s", string(originalBody))
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Final response body: %s", string(body))
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Final response body length: %d", len(body))

			// 直接使用DoResponse中已经转换好的响应体，不依赖重新读取
			// 这是一个临时解决方案，确保响应能正确返�?
			if len(body) == 0 {
				logger.LogError(c.Request.Context(), "[SUCHUANG] Final response body is empty after reading")
				// 尝试直接从原始响应中解析速创格式并转�?
				var suchuangResp struct {
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(originalBody, &suchuangResp); err == nil && suchuangResp.Data != nil {
					// 创建一个简单的OpenAI响应格式
					var openAIResp struct {
						ID      string `json:"id"`
						Object  string `json:"object"`
						Created int64  `json:"created"`
						Model   string `json:"model"`
						Choices []struct {
							Message struct {
								Content string `json:"content"`
								Role    string `json:"role"`
							} `json:"message"`
						} `json:"choices"`
					}

					// 尝试解析data字段为OpenAI响应
					if err := json.Unmarshal(suchuangResp.Data, &openAIResp); err == nil {
						// 如果解析成功，使用它
						body, _ = json.Marshal(openAIResp)
					} else {
						// 否则，将data字段作为content返回
						openAIResp.ID = fmt.Sprintf("chatcmpl-%d", time.Now().Unix())
						openAIResp.Object = "chat.completion"
						openAIResp.Created = time.Now().Unix()
						openAIResp.Model = info.OriginModelName
						openAIResp.Choices = []struct {
							Message struct {
								Content string `json:"content"`
								Role    string `json:"role"`
							} `json:"message"`
						}{{
							Message: struct {
								Content string `json:"content"`
								Role    string `json:"role"`
							}{Content: string(suchuangResp.Data), Role: "assistant"},
						}}
						body, _ = json.Marshal(openAIResp)
					}
					logger.LogDebug(c.Request.Context(), "[SUCHUANG] Fallback response body: %s", string(body))
					logger.LogDebug(c.Request.Context(), "[SUCHUANG] Fallback response body length: %d", len(body))
				}
			}

			// 使用Gin的c.Data方法直接返回转换后的响应�?
			logger.LogDebug(c.Request.Context(), "[SUCHUANG] Returning response with length: %d", len(body))
			c.Data(http.StatusOK, "application/json", body)
		} else {
			// 其他类型，直接序列化为JSON返回
			respBody, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				taskErr = service.TaskErrorWrapper(jsonErr, "marshal_response_failed", http.StatusInternalServerError)
				return
			}
			// 使用Gin的c.Data方法直接返回响应�?
			c.Data(http.StatusOK, "application/json", respBody)
		}

		// 图片生成请求已经完成，不需要继续处�?
		info.ConsumeQuota = true
		return nil
	} else {
		// 对于其他渠道，使用TaskAdaptor接口的实�?
		requestBody, err = adaptor.BuildRequestBody(c, info)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "build_request_failed", http.StatusInternalServerError)
			return
		}

		resp, err = adaptor.DoRequest(c, info, requestBody)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
			return
		}
	}

	// handle response status
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		taskErr = service.TaskErrorWrapper(errors.New(string(responseBody)), "fail_to_fetch_task", resp.StatusCode)
		return
	}

	// 检查是否是图片生成请求，如果是，直接返回完整响�?
	if strings.HasPrefix(c.Request.RequestURI, "/v1/images") {
		// 读取响应�?
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			taskErr = service.TaskErrorWrapper(readErr, "read_response_body_failed", http.StatusInternalServerError)
			return
		}
		// 使用Gin的c.Data方法直接返回响应�?
		c.Data(http.StatusOK, "application/json", body)
		// 图片生成请求已经完成，不需要继续处�?
		info.ConsumeQuota = true
		return nil
	}

	// 处理其他请求类型
	defer func() {
		// release quota
		if info.ConsumeQuota && taskErr == nil {

			err := service.PostConsumeQuota(info, quota, 0, true)
			if err != nil {
				common.SysLog("error consuming token remain quota: " + err.Error())
			}
			if quota != 0 {
				tokenName := c.GetString("token_name")
				//gRatio := groupRatio
				//if hasUserGroupRatio {
				//	gRatio = userGroupRatio
				//}
				logContent := fmt.Sprintf("操作 %s", info.Action)
				// FIXME: 临时修补，支持任务仅按次计费
				if common.StringsContains(constant.TaskPricePatches, modelName) {
					logContent = fmt.Sprintf("%s，按次计费", logContent)
				} else {
					if len(info.PriceData.OtherRatios) > 0 {
						var contents []string
						for key, ra := range info.PriceData.OtherRatios {
							if 1.0 != ra {
								contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
							}
						}
						if len(contents) > 0 {
							logContent = fmt.Sprintf("%s, 计算参数�?s", logContent, strings.Join(contents, ", "))
						}
					}
				}
				other := make(map[string]interface{})
				if c != nil && c.Request != nil && c.Request.URL != nil {
					other["request_path"] = c.Request.URL.Path
				}
				other["model_price"] = modelPrice
				other["group_ratio"] = groupRatio
				if hasUserGroupRatio {
					other["user_group_ratio"] = userGroupRatio
				}
				model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
					ChannelId: info.ChannelId,
					ModelName: modelName,
					TokenName: tokenName,
					Quota:     quota,
					Content:   logContent,
					TokenId:   info.TokenId,
					Group:     info.UsingGroup,
					Other:     other,
				})
				model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
				model.UpdateChannelUsedQuota(info.ChannelId, quota)
			}
		}
	}()

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		return
	}

	// 检查响应体是否已被修改（如在DoResponse中已设置�?
	if resp != nil && resp.Body != nil {
		// 读取修改后的响应�?
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr == nil && len(respBody) > 0 {
			// 直接返回响应�?
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.Write(respBody)
			info.ConsumeQuota = true
			return nil
		}
		// 重置响应体以便后续处�?
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	info.ConsumeQuota = true
	// insert task
	task := model.InitTask(platform, info)
	task.TaskID = taskID
	task.Quota = quota
	task.Data = taskData
	task.Action = info.Action
	err = task.Insert()
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "insert_task_failed", http.StatusInternalServerError)
		return
	}

	// 检查是否是OpenAI兼容的视频API请求
	if strings.HasPrefix(c.Request.RequestURI, "/v1/videos") {
		// 为视频API返回OpenAI格式的响�?
		openAIVideo := dto.NewOpenAIVideo()
		openAIVideo.ID = taskID
		openAIVideo.Status = "queued"                                           // 新提交的任务默认为queued状�?
		openAIVideo.CreatedAt = time.Now().UnixNano() / int64(time.Millisecond) // 当前时间戳（毫秒�?
		openAIVideo.Model = "sora-2"                                            // 根据实际情况设置模型名称

		// 获取用户输入的秒�?
		var seconds string = "10" // 默认视频时长
		if req, err := relaycommon.GetTaskRequest(c); err == nil {
			if req.Seconds != "" {
				seconds = req.Seconds
			} else if req.Duration != 0 {
				seconds = strconv.Itoa(req.Duration)
			}
		}
		openAIVideo.Seconds = seconds

		respBody, err := common.Marshal(openAIVideo)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
			return
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		_, err = io.Copy(c.Writer, bytes.NewBuffer(respBody))
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
			return
		}
	} else {
		// 其他API继续返回原有的包装格�?
		response := map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"data": map[string]interface{}{
				"id": taskID,
			},
		}

		respBody, err := json.Marshal(response)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
			return
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		_, err = io.Copy(c.Writer, bytes.NewBuffer(respBody))
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
			return
		}
	}

	return nil
}

var fetchRespBuilders = map[int]func(c *gin.Context) (respBody []byte, taskResp *dto.TaskError){
	relayconstant.RelayModeSunoFetchByID:  sunoFetchByIDRespBodyBuilder,
	relayconstant.RelayModeSunoFetch:      sunoFetchRespBodyBuilder,
	relayconstant.RelayModeVideoFetchByID: videoFetchByIDRespBodyBuilder,
}

func RelayTaskFetch(c *gin.Context, relayMode int) (taskResp *dto.TaskError) {
	respBuilder, ok := fetchRespBuilders[relayMode]
	if !ok {
		taskResp = service.TaskErrorWrapperLocal(errors.New("invalid_relay_mode"), "invalid_relay_mode", http.StatusBadRequest)
	}

	respBody, taskErr := respBuilder(c)
	if taskErr != nil {
		return taskErr
	}
	if len(respBody) == 0 {
		respBody = []byte("{\"code\":\"success\",\"data\":null}")
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := io.Copy(c.Writer, bytes.NewBuffer(respBody))
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
		return
	}
	return
}

func sunoFetchRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	userId := c.GetInt("id")
	var condition = struct {
		IDs    []any  `json:"ids"`
		Action string `json:"action"`
	}{}
	err := c.BindJSON(&condition)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
		return
	}
	var tasks []any
	if len(condition.IDs) > 0 {
		taskModels, err := model.GetByTaskIds(userId, condition.IDs)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "get_tasks_failed", http.StatusInternalServerError)
			return
		}
		for _, task := range taskModels {
			tasks = append(tasks, TaskModel2Dto(task))
		}
	} else {
		tasks = make([]any, 0)
	}
	respBody, err = json.Marshal(dto.TaskResponse[[]any]{
		Code: "success",
		Data: tasks,
	})
	return
}

func sunoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("id")
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	respBody, err = json.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	return
}

func videoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("task_id")
	if taskId == "" {
		taskId = c.GetString("task_id")
	}
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	func() {
		channelModel, err2 := model.GetChannelById(originTask.ChannelId, true)
		if err2 != nil {
			return
		}
		if channelModel.Type != constant.ChannelTypeVertexAi && channelModel.Type != constant.ChannelTypeGemini {
			return
		}
		baseURL := constant.ChannelBaseURLs[channelModel.Type]
		if channelModel.GetBaseURL() != "" {
			baseURL = channelModel.GetBaseURL()
		}
		adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
		if adaptor == nil {
			return
		}
		resp, err2 := adaptor.FetchTask(baseURL, channelModel.Key, map[string]any{
			"task_id": originTask.TaskID,
			"action":  originTask.Action,
		})
		if err2 != nil || resp == nil {
			return
		}
		defer resp.Body.Close()
		body, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return
		}
		ti, err2 := adaptor.ParseTaskResult(body)
		if err2 == nil && ti != nil {
			if ti.Status != "" {
				originTask.Status = model.TaskStatus(ti.Status)
			}
			if ti.Progress != "" {
				originTask.Progress = ti.Progress
			}
			// 根据任务状态处理Reason和URL字段
			if originTask.Status == model.TaskStatusFailure && ti.Reason != "" {
				originTask.FailReason = ti.Reason
			} else if originTask.Status == model.TaskStatusSuccess {
				// 任务成功时，将相关URL填入FailReason
				if ti.Reason != "" {
					originTask.FailReason = ti.Reason // 优先使用Reason字段（通常是remote_url�?
				} else if ti.RemoteUrl != "" {
					originTask.FailReason = ti.RemoteUrl // 使用remote_url
				} else if ti.Url != "" {
					originTask.FailReason = ti.Url // 使用url字段（video_url�?
				} else {
					// 尝试从任务data字段中解析URL（处理实时响应中URL缺失的情况）
					var soraResp struct {
						Data struct {
							RemoteURL   string `json:"remote_url"`
							TransferURL string `json:"transfer_url"`
							URL         string `json:"url"`
						} `json:"data"`
					}
					if err := json.Unmarshal(originTask.Data, &soraResp); err == nil {
						if soraResp.Data.RemoteURL != "" {
							originTask.FailReason = soraResp.Data.RemoteURL
						} else if soraResp.Data.TransferURL != "" {
							originTask.FailReason = soraResp.Data.TransferURL
						} else if soraResp.Data.URL != "" {
							originTask.FailReason = soraResp.Data.URL
						}
					}
				}
			}

			_ = originTask.Update()
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			format := "mp4"
			if respObj, ok := raw["response"].(map[string]any); ok {
				if vids, ok := respObj["videos"].([]any); ok && len(vids) > 0 {
					if v0, ok := vids[0].(map[string]any); ok {
						if mt, ok := v0["mimeType"].(string); ok && mt != "" {
							if strings.Contains(mt, "mp4") {
								format = "mp4"
							} else {
								format = mt
							}
						}
					}
				}
			}
			status := "processing"
			switch originTask.Status {
			case model.TaskStatusSuccess:
				status = "succeeded"
			case model.TaskStatusFailure:
				status = "failed"
			case model.TaskStatusQueued, model.TaskStatusSubmitted:
				status = "queued"
			}
			if !strings.HasPrefix(c.Request.RequestURI, "/v1/videos/") {
				out := map[string]any{
					"error":    nil,
					"format":   format,
					"metadata": nil,
					"status":   status,
					"task_id":  originTask.TaskID,
					"url":      originTask.FailReason,
				}
				respBody, _ = json.Marshal(dto.TaskResponse[any]{
					Code: "success",
					Data: out,
				})
			}
		}
	}()

	if len(respBody) != 0 {
		return
	}

	if strings.HasPrefix(c.Request.RequestURI, "/v1/videos/") {
		adaptor := GetTaskAdaptor(originTask.Platform)
		if adaptor == nil {
			taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("invalid channel id: %d", originTask.ChannelId), "invalid_channel_id", http.StatusBadRequest)
			return
		}
		if converter, ok := adaptor.(channel.OpenAIVideoConverter); ok {
			openAIVideoData, err := converter.ConvertToOpenAIVideo(originTask)
			if err != nil {
				taskResp = service.TaskErrorWrapper(err, "convert_to_openai_video_failed", http.StatusInternalServerError)
				return
			}
			respBody = openAIVideoData
			return
		}
		taskResp = service.TaskErrorWrapperLocal(errors.New(fmt.Sprintf("not_implemented:%s", originTask.Platform)), "not_implemented", http.StatusNotImplemented)
		return
	}
	respBody, err = json.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

func TaskModel2Dto(task *model.Task) *dto.TaskDto {
	return &dto.TaskDto{
		TaskID:     task.TaskID,
		Action:     task.Action,
		Status:     string(task.Status),
		FailReason: task.FailReason,
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		Progress:   task.Progress,
		Data:       task.Data,
	}
}
