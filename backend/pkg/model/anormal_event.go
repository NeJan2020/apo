package model

type AnormalType int

// AnormalType 所有可分类的异常平铺到最外层,便于前端过滤
const (
	AnormalTypeUnknown AnormalType = iota
	AnormalTypeAlertApp
	AnormalTypeAlertContainer
	AnormalTypeAlertInfra
	AnormalTypeAlertNet

	AnormalTypeMutation

	AnormalTypeError
)

// AnormalEvent 存储通用的异常事件,用于告警分析时汇总各种类型告警,统一返回
type AnormalEvent struct {
	// 事件状态更新时间戳
	UpdateTSs []AnormalUpdateTS `json:"updateTSs"`
	// 异常类型
	AnormalType AnormalType `json:"anormalType"`
	// 受异常影响的端点
	ImpactEndpoints []AnormalEventDetail `json:"impactEndpoints"`
}

type AnormalUpdateTS struct {
	AnormalStatus Status
	Timestamp     int64
}

type AnormalEventDetail struct {
	EndpointKey

	// 影响的实例
	AlertObject string `json:"alertObject"`

	// 粗略的影响描述
	AlertReason string `json:"alertReason"`

	// 不同时间点的异常信息
	AlertMessage map[int64]string `json:"alertMessage"`
}

type EndpointKey struct {
	ServiceName string `json:"serviceName"`
	ContentKey  string `json:"contentKey"`
}
