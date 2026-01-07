package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"xunkecloudAPI/common"
	"xunkecloudAPI/constant"
	"xunkecloudAPI/dto"
	"xunkecloudAPI/logger"
	relaycommon "xunkecloudAPI/relay/common"
	"xunkecloudAPI/relay/helper"
	"xunkecloudAPI/service"
	"xunkecloudAPI/setting/model_setting"
	"xunkecloudAPI/types"

	"github.com/gin-gonic/gin"
)

func ImageHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	var requestBody io.Reader

	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		body, err := common.GetRequestBody(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		requestBody = bytes.NewBuffer(body)
	} else {
		convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed)
		}

		switch convertedRequest.(type) {
		case *bytes.Buffer:
			requestBody = convertedRequest.(io.Reader)
		default:
			jsonData, err := common.Marshal(convertedRequest)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			// apply param override
			if len(info.ParamOverride) > 0 {
				jsonData, err = relaycommon.ApplyParamOverride(jsonData, info.ParamOverride)
				if err != nil {
					return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
				}
			}

			if common.DebugEnabled {
				logger.LogDebug(c, fmt.Sprintf("image request body: %s", string(jsonData)))
			}
			requestBody = bytes.NewBuffer(jsonData)
		}
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	var httpResp *http.Response
	// 检查返回的响应类型
	if resp != nil {
		// 尝试将响应转换为*http.Response类型
		if response, ok := resp.(*http.Response); ok {
			httpResp = response
			info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
			if httpResp.StatusCode != http.StatusOK {
				if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
					// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
					httpResp.StatusCode = http.StatusOK
				} else {
					newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
					// reset status code 重置状态码
					service.ResetStatusCode(newAPIError, statusCodeMappingStr)
					return newAPIError
				}
			}
		} else {
			// 如果返回的不是*http.Response类型，可能是已经处理好的数据
			// 将数据转换为JSON并写入响应体
			jsonData, err := json.Marshal(resp)
			if err != nil {
				return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
			}

			// 创建一个虚拟的HTTP响应对象
			httpResp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(jsonData)),
				Header:     make(http.Header),
			}
			httpResp.Header.Set("Content-Type", "application/json")

			// 将响应写入客户端
			c.Data(http.StatusOK, "application/json", jsonData)
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	if usage.(*dto.Usage).TotalTokens == 0 {
		usage.(*dto.Usage).TotalTokens = int(request.N)
	}
	if usage.(*dto.Usage).PromptTokens == 0 {
		usage.(*dto.Usage).PromptTokens = int(request.N)
	}

	quality := "standard"
	if request.Quality == "hd" {
		quality = "hd"
	}

	var logContent string

	if len(request.Size) > 0 {
		logContent = fmt.Sprintf("大小 %s, 品质 %s, 张数 %d", request.Size, quality, request.N)
	}

	// 如果是速创渠道，需要将响应体写回客户端
	if info.ChannelType == constant.ChannelTypeSuchuang && httpResp != nil && httpResp.Body != nil {
		respBody, _ := io.ReadAll(httpResp.Body)
		if len(respBody) > 0 {
			c.Data(http.StatusOK, "application/json", respBody)
		}
	}

	postConsumeQuota(c, info, usage.(*dto.Usage), logContent)
	return nil
}
