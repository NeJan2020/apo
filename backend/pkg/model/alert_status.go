package model

const (
	DelaySourceAlert    = "delaySource"
	InfrastructureAlert = "infrastructureStatus"
	NetAlert            = "netStatus"
	K8sEventAlert       = "k8sStatus"
	REDMetricsAlert     = "REDStatus"
	LogMetricsAlert     = "logsStatus"
)

type AlertStatusPROM struct {
}

type AlertStatusCH struct {
	REDMetricsStatus     string `json:"REDStatus"`            // RED指标告警
	LogMetricsStatus     string `json:"logsStatus"`           // 日志指标告警
	InfrastructureStatus string `json:"infrastructureStatus"` // 基础设施告警
	NetStatus            string `json:"netStatus"`            // 网络告警
	K8sStatus            string `json:"k8sStatus"`            // K8s状态告警
}

var NORMAL_ALERT_STATUS = AlertStatus{
	AlertStatusCH: AlertStatusCH{
		InfrastructureStatus: STATUS_NORMAL,
		NetStatus:            STATUS_NORMAL,
		K8sStatus:            STATUS_NORMAL,
		REDMetricsStatus:     STATUS_NORMAL,
		LogMetricsStatus:     STATUS_NORMAL,
	},
	AlertStatusPROM: AlertStatusPROM{},
}

// AlertStatus 告警状态
// 按数据来源分组只是为了统一管理
// 目前没有依赖于Prometheus数据的告警
type AlertStatus struct {
	// AlertStatusCH 基于clickhouse数据计算的告警
	AlertStatusCH
	// AlertStatusPROM 基于prometheus数据计算的告警
	AlertStatusPROM
}

type AlertReason map[string][]AlertDetail

type AlertDetail struct {
	Timestamp    int64  `json:"timestamp"`
	AlertObject  string `json:"alertObject"`
	AlertReason  string `json:"alertReason"`
	AlertMessage string `json:"alertMessage"`
}

func (r AlertReason) Add(key string, detail AlertDetail) {
	if len(key) == 0 {
		return
	}

	if r == nil {
		return
	}

	details, find := r[key]
	if !find {
		r[key] = []AlertDetail{detail}
		return
	}

	r[key] = append(details, detail)
}
