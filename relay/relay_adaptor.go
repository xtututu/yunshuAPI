package relay

import (
	"strconv"

	"yishangyunApi/constant"
	"yishangyunApi/relay/channel"
	"yishangyunApi/relay/channel/ali"
	"yishangyunApi/relay/channel/aws"
	"yishangyunApi/relay/channel/baidu"
	"yishangyunApi/relay/channel/baidu_v2"
	"yishangyunApi/relay/channel/claude"
	"yishangyunApi/relay/channel/cloudflare"
	"yishangyunApi/relay/channel/cohere"
	"yishangyunApi/relay/channel/coze"
	"yishangyunApi/relay/channel/deepseek"
	"yishangyunApi/relay/channel/dify"
	"yishangyunApi/relay/channel/gemini"
	"yishangyunApi/relay/channel/jimeng"
	"yishangyunApi/relay/channel/jina"
	"yishangyunApi/relay/channel/minimax"
	"yishangyunApi/relay/channel/mistral"
	"yishangyunApi/relay/channel/mokaai"
	"yishangyunApi/relay/channel/moonshot"
	"yishangyunApi/relay/channel/ollama"
	"yishangyunApi/relay/channel/openai"
	"yishangyunApi/relay/channel/palm"
	"yishangyunApi/relay/channel/perplexity"
	"yishangyunApi/relay/channel/replicate"
	"yishangyunApi/relay/channel/siliconflow"
	"yishangyunApi/relay/channel/submodel"
	taskali "yishangyunApi/relay/channel/task/ali"
	taskdoubao "yishangyunApi/relay/channel/task/doubao"
	taskGemini "yishangyunApi/relay/channel/task/gemini"
	taskjimeng "yishangyunApi/relay/channel/task/jimeng"
	"yishangyunApi/relay/channel/task/kling"
	tasksora "yishangyunApi/relay/channel/task/sora"
	"yishangyunApi/relay/channel/task/suno"
	taskvertex "yishangyunApi/relay/channel/task/vertex"
	taskVidu "yishangyunApi/relay/channel/task/vidu"
	"yishangyunApi/relay/channel/tencent"
	"yishangyunApi/relay/channel/vertex"
	"yishangyunApi/relay/channel/volcengine"
	"yishangyunApi/relay/channel/xai"
	"yishangyunApi/relay/channel/xunfei"
	"yishangyunApi/relay/channel/zhipu"
	"yishangyunApi/relay/channel/zhipu_4v"
	"github.com/gin-gonic/gin"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeBaidu:
		return &baidu.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipu:
		return &zhipu.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypePerplexity:
		return &perplexity.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeMokaAI:
		return &mokaai.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeBaiduV2:
		return &baidu_v2.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXinference:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
	case constant.APITypeCoze:
		return &coze.Adaptor{}
	case constant.APITypeJimeng:
		return &jimeng.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{} // Moonshot uses Claude API
	case constant.APITypeSubmodel:
		return &submodel.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	case constant.APITypeReplicate:
		return &replicate.Adaptor{}
	}
	return nil
}

func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
	channelType := c.GetInt("channel_type")
	if channelType > 0 {
		return constant.TaskPlatform(strconv.Itoa(channelType))
	}
	return constant.TaskPlatform(c.GetString("platform"))
}

func GetTaskAdaptor(platform constant.TaskPlatform) channel.TaskAdaptor {
	switch platform {
	//case constant.APITypeAIProxyLibrary:
	//	return &aiproxy.Adaptor{}
	case constant.TaskPlatformSuno:
		return &suno.TaskAdaptor{}
	}
	if channelType, err := strconv.ParseInt(string(platform), 10, 64); err == nil {
		switch channelType {
		case constant.ChannelTypeAli:
			return &taskali.TaskAdaptor{}
		case constant.ChannelTypeKling:
			return &kling.TaskAdaptor{}
		case constant.ChannelTypeJimeng:
			return &taskjimeng.TaskAdaptor{}
		case constant.ChannelTypeVertexAi:
			return &taskvertex.TaskAdaptor{}
		case constant.ChannelTypeVidu:
			return &taskVidu.TaskAdaptor{}
		case constant.ChannelTypeDoubaoVideo:
			return &taskdoubao.TaskAdaptor{}
		case constant.ChannelTypeSora, constant.ChannelTypeOpenAI:
			return &tasksora.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		}
	}
	return nil
}
