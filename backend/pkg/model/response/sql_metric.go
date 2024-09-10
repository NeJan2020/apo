package response

import "github.com/CloudDetail/apo/backend/pkg/repository/prometheus"

type GetSQLMetricsResponse struct {

	// TODO 限制返回的SQLOperationDetail总数
	SQLOperationDetails []SQLOperationDetail `json:"sqlOperationDetails"`
}

type SQLOperationDetail struct {
	prometheus.SQLKey

	Latency   TempChartObject `json:"latency"`
	ErrorRate TempChartObject `json:"errorRate"`
	// FIXME Tps 名称为tps,实际为每分钟请求数
	Tps TempChartObject `json:"tps"`
}