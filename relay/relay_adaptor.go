package relay

import (
	"strconv"

	"yunshuAPI/constant"
	"yunshuAPI/relay/channel"
	"yunshuAPI/relay/channel/ali"
	"yunshuAPI/relay/channel/aws"
	"yunshuAPI/relay/channel/baidu"
	"yunshuAPI/relay/channel/baidu_v2"
	"yunshuAPI/relay/channel/claude"
	"yunshuAPI/relay/channel/cloudflare"
	"yunshuAPI/relay/channel/cohere"
	"yunshuAPI/relay/channel/coze"
	"yunshuAPI/relay/channel/deepseek"
	"yunshuAPI/relay/channel/dify"
	"yunshuAPI/relay/channel/gemini"
	"yunshuAPI/relay/channel/jimeng"
	"yunshuAPI/relay/channel/jina"
	"yunshuAPI/relay/channel/kieai"
	"yunshuAPI/relay/channel/minimax"
	"yunshuAPI/relay/channel/mistral"
	"yunshuAPI/relay/channel/mokaai"
	"yunshuAPI/relay/channel/moonshot"
	"yunshuAPI/relay/channel/ollama"
	"yunshuAPI/relay/channel/openai"
	"yunshuAPI/relay/channel/palm"
	"yunshuAPI/relay/channel/perplexity"
	"yunshuAPI/relay/channel/replicate"
	"yunshuAPI/relay/channel/siliconflow"
	"yunshuAPI/relay/channel/submodel"
	"yunshuAPI/relay/channel/suchuang"
	taskali "yunshuAPI/relay/channel/task/ali"
	taskdoubao "yunshuAPI/relay/channel/task/doubao"
	taskGemini "yunshuAPI/relay/channel/task/gemini"
	taskjimeng "yunshuAPI/relay/channel/task/jimeng"
	"yunshuAPI/relay/channel/task/kling"
	tasksora "yunshuAPI/relay/channel/task/sora"
	tasksorag "yunshuAPI/relay/channel/task/sora-g"
	tasksoras "yunshuAPI/relay/channel/task/sora-s"
	"yunshuAPI/relay/channel/task/suno"
	taskvertex "yunshuAPI/relay/channel/task/vertex"
	"yunshuAPI/relay/channel/task/vidu"
	"yunshuAPI/relay/channel/tencent"
	"yunshuAPI/relay/channel/vertex"
	"yunshuAPI/relay/channel/volcengine"
	"yunshuAPI/relay/channel/xai"
	"yunshuAPI/relay/channel/xunfei"
	"yunshuAPI/relay/channel/zhipu"
	"yunshuAPI/relay/channel/zhipu_4v"
	"yunshuAPI/relay/channel/yushu"

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
	case constant.APITypeSuchuang:
		return &suchuang.Adaptor{}
	case constant.ChannelTypeKieai:
		return &kieai.Adaptor{}
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
			return &vidu.TaskAdaptor{}
		case constant.ChannelTypeDoubaoVideo:
			return &taskdoubao.TaskAdaptor{}
		case constant.ChannelTypeSora, constant.ChannelTypeOpenAI:
			return &tasksora.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		case constant.ChannelTypeSoraG:
			return &tasksorag.TaskAdaptor{}
		case constant.ChannelTypeSoraS:
			return &tasksoras.TaskAdaptor{}
		case constant.ChannelTypeSuchuang:
			return &suchuang.TaskAdaptor{}
		case constant.ChannelTypeKieai:
			return &kieai.TaskAdaptor{}
		case constant.ChannelTypeYushu:
			return &yushu.TaskAdaptor{}
		}
	}
	return nil
}
