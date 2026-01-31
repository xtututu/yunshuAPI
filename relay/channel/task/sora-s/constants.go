package soras

// 常量定义
const (
	// 渠道类型
	ChannelType = "sora-s"
	
	// API端点
	APIEndpointSora2     = "https://api.wuyinkeji.com/api/sora2-new/submit"
	APIEndpointSora2Pro  = "https://api.wuyinkeji.com/api/sora2pro/submit"
	APIEndpointDetail    = "https://api.wuyinkeji.com/api/sora2/detail"
	
	// 模型名称
	ModelSora2      = "sora-2"
	ModelSora2HD    = "sora-2-hd"
	ModelSora2Pro   = "sora-2-pro"
	
	// 任务状态
	TaskStatusPending   = 0 // 待处理
	TaskStatusProcessing = 1 // 处理中
	TaskStatusSuccess   = 2 // 成功
	TaskStatusFailed    = 3 // 失败
)