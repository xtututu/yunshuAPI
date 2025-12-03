package constant

const (
	ChannelTypeUnknown        = 0
	ChannelTypeOpenAI         = 1
	ChannelTypeMidjourney     = 2
	ChannelTypeAzure          = 3
	ChannelTypeOllama         = 4
	ChannelTypeMidjourneyPlus = 5
	ChannelTypeOpenAIMax      = 6
	ChannelTypeOhMyGPT        = 7
	ChannelTypeCustom         = 8
	ChannelTypeAILS           = 9
	ChannelTypeAIProxy        = 10
	ChannelTypePaLM           = 11
	ChannelTypeAPI2GPT        = 12
	ChannelTypeAIGC2D         = 13
	ChannelTypeAnthropic      = 14
	ChannelTypeBaidu          = 15
	ChannelTypeZhipu          = 16
	ChannelTypeAli            = 17
	ChannelTypeXunfei         = 18
	ChannelType360            = 19
	ChannelTypeOpenRouter     = 20
	ChannelTypeAIProxyLibrary = 21
	ChannelTypeFastGPT        = 22
	ChannelTypeTencent        = 23
	ChannelTypeGemini         = 24
	ChannelTypeMoonshot       = 25
	ChannelTypeZhipu_v4       = 26
	ChannelTypePerplexity     = 27
	ChannelTypeLingYiWanWu    = 31
	ChannelTypeAws            = 33
	ChannelTypeCohere         = 34
	ChannelTypeMiniMax        = 35
	ChannelTypeSunoAPI        = 36
	ChannelTypeDify           = 37
	ChannelTypeJina           = 38
	ChannelCloudflare         = 39
	ChannelTypeSiliconFlow    = 40
	ChannelTypeVertexAi       = 41
	ChannelTypeMistral        = 42
	ChannelTypeDeepSeek       = 43
	ChannelTypeMokaAI         = 44
	ChannelTypeVolcEngine     = 45
	ChannelTypeBaiduV2        = 46
	ChannelTypeXinference     = 47
	ChannelTypeXai            = 48
	ChannelTypeCoze           = 49
	ChannelTypeKling          = 50
	ChannelTypeJimeng         = 51
	ChannelTypeVidu           = 52
	ChannelTypeSubmodel       = 53
	ChannelTypeDoubaoVideo    = 54
	ChannelTypeSora           = 55
	ChannelTypeReplicate      = 56
	ChannelTypeDummy          // this one is only for count, do not add any channel after this

)

var ChannelBaseURLs = []string{
	"",                        // 0
	"https://api.chatfire.cn", // 1
	"https://api.chatfire.cn", // 2
	"https://api.chatfire.cn", // 3
	"https://api.chatfire.cn", // 4
	"https://api.chatfire.cn", // 5
	"https://api.chatfire.cn", // 6
	"https://api.chatfire.cn", // 7
	"https://api.chatfire.cn", // 8
	"https://api.chatfire.cn", // 9
	"https://api.chatfire.cn", // 10
	"https://api.chatfire.cn", // 11
	"https://api.chatfire.cn", // 12
	"https://api.chatfire.cn", // 13
	"https://api.chatfire.cn", // 14
	"https://api.chatfire.cn", // 15
	"https://api.chatfire.cn", // 16
	"https://api.chatfire.cn", // 17
	"https://api.chatfire.cn", // 18
	"https://api.chatfire.cn", // 19
	"https://api.chatfire.cn", // 20
	"https://api.chatfire.cn", // 21
	"https://api.chatfire.cn", // 22
	"https://api.chatfire.cn", // 23
	"https://api.chatfire.cn", // 24
	"https://api.chatfire.cn", // 25
	"https://api.chatfire.cn", // 26
	"https://api.chatfire.cn", // 27
	"https://api.chatfire.cn", // 28
	"https://api.chatfire.cn", // 29
	"https://api.chatfire.cn", // 30
	"https://api.chatfire.cn", // 31
	"https://api.chatfire.cn", // 32
	"https://api.chatfire.cn", // 33
	"https://api.chatfire.cn", // 34
	"https://api.chatfire.cn", // 35
	"https://api.chatfire.cn", // 36
	"https://api.chatfire.cn", // 37
	"https://api.chatfire.cn", // 38
	"https://api.chatfire.cn", // 39
	"https://api.chatfire.cn", // 40
	"https://api.chatfire.cn", // 41
	"https://api.chatfire.cn", // 42
	"https://api.chatfire.cn", // 43
	"https://api.chatfire.cn", // 44
	"https://api.chatfire.cn", // 45
	"https://api.chatfire.cn", // 46
	"https://api.chatfire.cn", // 47
	"https://api.chatfire.cn", // 48
	"https://api.chatfire.cn", // 49
	"https://api.chatfire.cn", // 50
	"https://api.chatfire.cn", // 51
	"https://api.chatfire.cn", // 52
	"https://api.chatfire.cn", // 53
	"https://api.chatfire.cn", // 54
	"https://api.chatfire.cn", // 55
	"https://api.chatfire.cn", // 56
}

var ChannelTypeNames = map[int]string{
	ChannelTypeUnknown:        "Unknown",
	ChannelTypeOpenAI:         "OpenAI",
	ChannelTypeMidjourney:     "Midjourney",
	ChannelTypeAzure:          "Azure",
	ChannelTypeOllama:         "Ollama",
	ChannelTypeMidjourneyPlus: "MidjourneyPlus",
	ChannelTypeOpenAIMax:      "OpenAIMax",
	ChannelTypeOhMyGPT:        "OhMyGPT",
	ChannelTypeCustom:         "Custom",
	ChannelTypeAILS:           "AILS",
	ChannelTypeAIProxy:        "AIProxy",
	ChannelTypePaLM:           "PaLM",
	ChannelTypeAPI2GPT:        "API2GPT",
	ChannelTypeAIGC2D:         "AIGC2D",
	ChannelTypeAnthropic:      "Anthropic",
	ChannelTypeBaidu:          "Baidu",
	ChannelTypeZhipu:          "Zhipu",
	ChannelTypeAli:            "Ali",
	ChannelTypeXunfei:         "Xunfei",
	ChannelType360:            "360",
	ChannelTypeOpenRouter:     "OpenRouter",
	ChannelTypeAIProxyLibrary: "AIProxyLibrary",
	ChannelTypeFastGPT:        "FastGPT",
	ChannelTypeTencent:        "Tencent",
	ChannelTypeGemini:         "Gemini",
	ChannelTypeMoonshot:       "Moonshot",
	ChannelTypeZhipu_v4:       "ZhipuV4",
	ChannelTypePerplexity:     "Perplexity",
	ChannelTypeLingYiWanWu:    "LingYiWanWu",
	ChannelTypeAws:            "AWS",
	ChannelTypeCohere:         "Cohere",
	ChannelTypeMiniMax:        "MiniMax",
	ChannelTypeSunoAPI:        "SunoAPI",
	ChannelTypeDify:           "Dify",
	ChannelTypeJina:           "Jina",
	ChannelCloudflare:         "Cloudflare",
	ChannelTypeSiliconFlow:    "SiliconFlow",
	ChannelTypeVertexAi:       "VertexAI",
	ChannelTypeMistral:        "Mistral",
	ChannelTypeDeepSeek:       "DeepSeek",
	ChannelTypeMokaAI:         "MokaAI",
	ChannelTypeVolcEngine:     "VolcEngine",
	ChannelTypeBaiduV2:        "BaiduV2",
	ChannelTypeXinference:     "Xinference",
	ChannelTypeXai:            "xAI",
	ChannelTypeCoze:           "Coze",
	ChannelTypeKling:          "Kling",
	ChannelTypeJimeng:         "Jimeng",
	ChannelTypeVidu:           "Vidu",
	ChannelTypeSubmodel:       "Submodel",
	ChannelTypeDoubaoVideo:    "DoubaoVideo",
	ChannelTypeSora:           "Sora",
	ChannelTypeReplicate:      "Replicate",
}

func GetChannelTypeName(channelType int) string {
	if name, ok := ChannelTypeNames[channelType]; ok {
		return name
	}
	return "Unknown"
}
