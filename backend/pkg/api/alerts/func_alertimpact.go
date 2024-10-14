package alerts

import (
	"errors"
	"net/http"
	"time"

	"github.com/CloudDetail/apo/backend/pkg/code"
	"github.com/CloudDetail/apo/backend/pkg/core"
	"github.com/CloudDetail/apo/backend/pkg/model"
	"github.com/CloudDetail/apo/backend/pkg/model/request"
	"github.com/CloudDetail/apo/backend/pkg/model/response"
	"github.com/CloudDetail/apo/backend/pkg/repository/clickhouse"
	"github.com/CloudDetail/apo/backend/pkg/repository/database"
	"github.com/CloudDetail/apo/backend/pkg/services/serviceoverview"
	"go.uber.org/zap"
)

// AlertImpact 获取告警数据的影响面
// @Summary 获取告警数据的影响面
// @Description 获取告警数据的影响面
// @Tags API.alerts
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param eventId query string true "查询告警事件ID"
// @Param startTime query uint64 true "查询开始时间"
// @Param endTime query uint64 true "查询结束时间"
// @Param step query int64 true "查询步长(us)"
// @Success 200 {object} []response.GetServiceEntryEndpointsResponse
// @Failure 400 {object} code.Failure
// @Router /api/alerts/impact [post]
func (h *handler) AlertImpact() core.HandlerFunc {
	return func(c core.Context) {
		req := new(request.AlertImpactRequest)
		if err := c.ShouldBindQuery(req); err != nil {
			c.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				code.Text(code.ParamBindError)).WithError(err),
			)
			return
		}

		entryNodes, err := h.alertService.AlertImpact(req.EventID, req.StartTime, req.EndTime)
		if err != nil {
			var vErr model.ErrAlertImpactMissingTag
			if errors.As(err, &vErr) {
				// 告警所需的关联信息不足
				// 提示告警中需要包含下列任意label组合
				c.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.AlertEventImpactMissingTag,
					code.Text(code.AlertEventImpactMissingTag)+vErr.CheckedTagGroups()).WithError(err),
				)
				return
			} else {
				// 查询失败
				c.AbortWithError(core.Error(
					http.StatusBadRequest,
					code.AlertEventImpactError,
					code.Text(code.AlertEventImpactError)).WithError(err),
				)
			}
		}

		// 填充EntryEndpoint信息
		resp := h.FillEntryNodeDetail(req, entryNodes)
		c.Payload(resp)
	}
}

// FillEntryNodeDetail 填充EntryEndpoint信息
// 复制于 backend/pkg/api/service/func_getserviceentryendpoints.go
func (h *handler) FillEntryNodeDetail(req *request.AlertImpactRequest, entryNodes []clickhouse.EntryNode) response.GetServiceEntryEndpointsResponse {
	result := make(map[string]*response.EntryInstanceData, 0)
	resp := response.GetServiceEntryEndpointsResponse{
		Status: model.STATUS_NORMAL,
		Data:   make([]*response.EntryInstanceData, 0),
	}

	threshold, err := h.serviceoverviewService.GetThreshold(database.GLOBAL, "", "")
	if err != nil {
		// 获取全局Threshold失败，使用默认值
		threshold = response.GetThresholdResponse{
			Latency:   5,
			ErrorRate: 5,
			Tps:       5,
			Log:       5,
		}
	}

	startTime := time.UnixMicro(req.StartTime)
	endTime := time.UnixMicro(req.EndTime)
	sortRule := serviceoverview.DODThreshold
	step := time.Duration(req.Step * 1000)

	for _, entryNode := range entryNodes {
		filter := serviceoverview.EndpointsFilter{
			ContainsSvcName:      entryNode.Service,
			ContainsEndpointName: entryNode.Endpoint,
			Namespace:            "",
		}
		endpointResps, err := h.serviceoverviewService.GetServicesEndPointData(startTime, endTime, step, filter, sortRule)
		if err != nil {
			continue
		}
		for _, endpointResp := range endpointResps {
			if serviceResp, found := result[endpointResp.ServiceName]; found {
				serviceResp.Namespaces = endpointResp.Namespaces
				serviceResp.EndpointCount += endpointResp.EndpointCount
				serviceResp.AddNamespaces(endpointResp.Namespaces)
			} else {
				result[endpointResp.ServiceName] = &response.EntryInstanceData{
					ServiceName:    endpointResp.ServiceName,
					Namespaces:     endpointResp.Namespaces,
					EndpointCount:  endpointResp.EndpointCount,
					ServiceDetails: endpointResp.ServiceDetails,
				}
			}

			for _, detail := range endpointResp.ServiceDetails {
				if detail.Latency.Ratio.DayOverDay != nil && *detail.Latency.Ratio.DayOverDay > threshold.Latency {
					resp.Status = model.STATUS_CRITICAL
				}
				if detail.Latency.Ratio.WeekOverDay != nil && *detail.Latency.Ratio.WeekOverDay > threshold.Latency {
					resp.Status = model.STATUS_CRITICAL
				}
				if detail.ErrorRate.Ratio.DayOverDay != nil && *detail.ErrorRate.Ratio.DayOverDay > threshold.ErrorRate {
					resp.Status = model.STATUS_CRITICAL
				}
				if detail.ErrorRate.Ratio.WeekOverDay != nil && *detail.ErrorRate.Ratio.WeekOverDay > threshold.ErrorRate {
					resp.Status = model.STATUS_CRITICAL
				}
			}
		}
	}

	serviceNames := make([]string, 0)
	for serviceName := range result {
		serviceNames = append(serviceNames, serviceName)
	}

	// 补全日志错误数等信息
	alertResps, err := h.serviceoverviewService.GetServicesAlert(startTime, endTime, step, serviceNames, nil)
	if err != nil {
		// 未能检查到状态,输出日志
		h.logger.Error("get entryEndpoint alert error", zap.Error(err))
	}
	for _, alertResp := range alertResps {
		if serviceResp, found := result[alertResp.ServiceName]; found {
			serviceResp.Logs = alertResp.Logs
			serviceResp.Timestamp = alertResp.Timestamp
			serviceResp.AlertStatus = alertResp.AlertStatus
			serviceResp.AlertReason = alertResp.AlertReason
		}

		if alertResp.Logs.Ratio.DayOverDay != nil && *alertResp.Logs.Ratio.DayOverDay > threshold.Log {
			resp.Status = model.STATUS_CRITICAL
		}
		if alertResp.Logs.Ratio.WeekOverDay != nil && *alertResp.Logs.Ratio.WeekOverDay > threshold.Log {
			resp.Status = model.STATUS_CRITICAL
		}
		if alertResp.AlertStatusCH.InfrastructureStatus == model.STATUS_CRITICAL ||
			alertResp.AlertStatusCH.NetStatus == model.STATUS_CRITICAL ||
			alertResp.AlertStatusCH.K8sStatus == model.STATUS_CRITICAL {
			resp.Status = model.STATUS_CRITICAL
		}
	}

	for _, endpointsResp := range result {
		resp.Data = append(resp.Data, endpointsResp)
	}
	return resp
}
