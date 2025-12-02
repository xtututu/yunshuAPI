package relay

import (
	"strconv"

	"xunkecloudAPI/constant"
	"xunkecloudAPI/relay/channel"
	"xunkecloudAPI/relay/channel/ali"
	"xunkecloudAPI/relay/channel/aws"
	"xunkecloudAPI/relay/channel/baidu"
	"xunkecloudAPI/relay/channel/baidu_v2"
	"xunkecloudAPI/relay/channel/claude"
	"xunkecloudAPI/relay/channel/cloudflare"
	"xunkecloudAPI/relay/channel/cohere"
	"xunkecloudAPI/relay/channel/coze"
	"xunkecloudAPI/relay/channel/deepseek"
	"xunkecloudAPI/relay/channel/dify"
	"xunkecloudAPI/relay/channel/gemini"
	"xunkecloudAPI/relay/channel/jimeng"
	"xunkecloudAPI/relay/channel/jina"
	"xunkecloudAPI/relay/channel/minimax"
	"xunkecloudAPI/relay/channel/mistral"
	"xunkecloudAPI/relay/channel/mokaai"
	"xunkecloudAPI/relay/channel/moonshot"
	"xunkecloudAPI/relay/channel/ollama"
	"xunkecloudAPI/relay/channel/openai"
	"xunkecloudAPI/relay/channel/palm"
	"xunkecloudAPI/relay/channel/perplexity"
	"xunkecloudAPI/relay/channel/replicate"
	"xunkecloudAPI/relay/channel/siliconflow"
	"xunkecloudAPI/relay/channel/submodel"
	taskali "xunkecloudAPI/relay/channel/task/ali"
	taskdoubao "xunkecloudAPI/relay/channel/task/doubao"
	taskGemini "xunkecloudAPI/relay/channel/task/gemini"
	taskjimeng "xunkecloudAPI/relay/channel/task/jimeng"
	"xunkecloudAPI/relay/channel/task/kling"
	tasksora "xunkecloudAPI/relay/channel/task/sora"
	"xunkecloudAPI/relay/channel/task/suno"
	taskvertex "xunkecloudAPI/relay/channel/task/vertex"
	taskVidu "xunkecloudAPI/relay/channel/task/vidu"
	"xunkecloudAPI/relay/channel/tencent"
	"xunkecloudAPI/relay/channel/vertex"
	"xunkecloudAPI/relay/channel/volcengine"
	"xunkecloudAPI/relay/channel/xai"
	"xunkecloudAPI/relay/channel/xunfei"
	"xunkecloudAPI/relay/channel/zhipu"
	"xunkecloudAPI/relay/channel/zhipu_4v"
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
