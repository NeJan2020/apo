package model

type AnormalType int

// AnormalType 所有可分类的异常平铺到最外层,便于前端过滤
const (
	AnormalTypeUnknown AnormalType = iota
	AnormalTypeAlertApp
	AnormalTypeAlertContainer
	AnormalTypeAlertInfra
	AnormalTypeAlertNet

	AnormalTypeExceptionJAVA
)

// AnormalEvent 存储通用的异常事件,用于告警分析时汇总各种类型告警,统一返回
type AnormalEvent struct {
	// 事件发生的时间戳
	Timestamp int64 `json:"timestamp"`
	// 异常类型
	AnormalType AnormalType `json:"anormalType"`
	// 受异常影响的端点
	ImpactEndpoints []AnormalEventDetail `json:"impactEndpoints"`
}

type AnormalEventDetail struct {
	EndpointKey

	// 影响的实例
	AlertObject string `json:"alertObject"`

	// 粗略的影响描述
	AlertReason string `json:"alertReason"`

	// 具体的事件信息
	AlertMessage string `json:"alertMessage"`
}

type EndpointKey struct {
	ServiceName string
	ContentKey  string
}
