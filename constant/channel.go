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
	ChannelTypeCloudflare     = 39
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
	ChannelTypeSoraG          = 57
	ChannelTypeSoraS          = 58
	ChannelTypeOpenAIS        = 59
	ChannelTypeSuchuang       = 60
	ChannelTypeKieai          = 61
	ChannelTypeDummy          // this one is only for count, do not add any channel after this

)

var ChannelBaseURLs = []string{
	"",                            // 0
	"https://api.vectorengine.ai", // 1
	"https://api.vectorengine.ai", // 2
	"https://api.vectorengine.ai", // 3
	"https://api.vectorengine.ai", // 4
	"https://api.vectorengine.ai", // 5
	"https://api.vectorengine.ai", // 6
	"https://api.vectorengine.ai", // 7
	"https://api.vectorengine.ai", // 8
	"https://api.vectorengine.ai", // 9
	"https://api.vectorengine.ai", // 10
	"https://api.vectorengine.ai", // 11
	"https://api.vectorengine.ai", // 12
	"https://api.vectorengine.ai", // 13
	"https://api.vectorengine.ai", // 14
	"https://api.vectorengine.ai", // 15
	"https://api.vectorengine.ai", // 16
	"https://api.vectorengine.ai", // 17
	"https://api.vectorengine.ai", // 18
	"https://api.vectorengine.ai", // 19
	"https://api.vectorengine.ai", // 20
	"https://api.vectorengine.ai", // 21
	"https://api.vectorengine.ai", // 22
	"https://api.vectorengine.ai", // 23
	"https://api.vectorengine.ai", // 24
	"https://api.vectorengine.ai", // 25
	"https://api.vectorengine.ai", // 26
	"https://api.vectorengine.ai", // 27
	"https://api.vectorengine.ai", // 28
	"https://api.vectorengine.ai", // 29
	"https://api.vectorengine.ai", // 30
	"https://api.vectorengine.ai", // 31
	"https://api.vectorengine.ai", // 32
	"https://api.vectorengine.ai", // 33
	"https://api.vectorengine.ai", // 34
	"https://api.vectorengine.ai", // 35
	"https://api.vectorengine.ai", // 36
	"https://api.vectorengine.ai", // 37
	"https://api.vectorengine.ai", // 38
	"https://api.vectorengine.ai", // 39
	"https://api.vectorengine.ai", // 40
	"https://api.vectorengine.ai", // 41
	"https://api.vectorengine.ai", // 42
	"https://api.vectorengine.ai", // 43
	"https://api.vectorengine.ai", // 44
	"https://api.vectorengine.ai", // 45
	"https://api.vectorengine.ai", // 46
	"https://api.vectorengine.ai", // 47
	"https://api.vectorengine.ai", // 48
	"https://api.vectorengine.ai", // 49
	"https://api.vectorengine.ai", // 50
	"https://api.vectorengine.ai", // 51
	"https://api.vectorengine.ai", // 52
	"https://api.vectorengine.ai", // 53
	"https://api.vectorengine.ai", // 54
	"https://api.vectorengine.ai", // 55
	"https://api.vectorengine.ai", // 56
	"https://grsai.dakka.com.cn",  // 57
	"https://api.wuyinkeji.com",   // 58
	"https://api.wuyinkeji.com",   // 59
	"https://api.wuyinkeji.com",   // 60
	"https://api.kie.ai",          // 61
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
	ChannelTypeCloudflare:     "Cloudflare",
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
	ChannelTypeSoraG:          "Sora-g",
	ChannelTypeSoraS:          "Sora-s",
	ChannelTypeOpenAIS:        "OpenAI-s",
	ChannelTypeSuchuang:       "Suchuang",
	ChannelTypeKieai:          "KieAI",
}

func GetChannelTypeName(channelType int) string {
	if name, ok := ChannelTypeNames[channelType]; ok {
		return name
	}
	return "Unknown"
}
