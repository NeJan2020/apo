package service

import (
	"fmt"
	"time"

	"github.com/CloudDetail/apo/backend/pkg/model"
	"github.com/CloudDetail/apo/backend/pkg/model/request"
	"github.com/CloudDetail/apo/backend/pkg/model/response"
	"github.com/CloudDetail/apo/backend/pkg/repository/polarisanalyzer"
	prom "github.com/CloudDetail/apo/backend/pkg/repository/prometheus"
	"github.com/CloudDetail/apo/backend/pkg/services/serviceoverview"
)

// GetDescendantRelevance implements Service.
func (s *service) GetDescendantRelevance(req *request.GetDescendantRelevanceRequest) ([]response.GetDescendantRelevanceResponse, error) {
	// 查询所有子孙节点
	nodes, err := s.chRepo.ListDescendantNodes(req)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return make([]response.GetDescendantRelevanceResponse, 0), nil
	}

	unsortedDescendant := make([]polarisanalyzer.LatencyRelevance, 0, len(nodes))
	var services, endpoints []string
	for _, node := range nodes {
		unsortedDescendant = append(unsortedDescendant, polarisanalyzer.LatencyRelevance{
			Service:  node.Service,
			Endpoint: node.Endpoint,
		})
		services = append(services, node.Service)
		endpoints = append(endpoints, node.Endpoint)
	}

	// 按延时相似度排序
	// sorted, unsorted, err :=
	sortResp, err := s.polRepo.SortDescendantByLatencyRelevance(
		req.StartTime, req.EndTime, prom.VecFromDuration(time.Duration(req.Step)*time.Microsecond),
		req.Service, req.Endpoint,
		unsortedDescendant,
	)

	var sortResult []polarisanalyzer.LatencyRelevance
	var sortType string
	if err != nil || sortResp == nil {
		sortResult = unsortedDescendant
		sortType = "net_failed"
	} else {
		sortResult = sortResp.SortedDescendant
		sortType = sortResp.DistanceType
		// 将未能排序成功的下游添加到descendants后(可能是没有北极星指标)
		for _, descendant := range sortResp.UnsortedDescendant {
			sortResult = append(sortResult, polarisanalyzer.LatencyRelevance{
				Service:  descendant.Service,
				Endpoint: descendant.Endpoint,
			})
		}
	}

	var resp []response.GetDescendantRelevanceResponse
	descendantStatus, err := s.queryDescendantDelaySource(services, endpoints, req.StartTime, req.EndTime)
	if err != nil {
		// TODO 添加日志,查询RED指标失败
	}
	for _, descendant := range sortResult {
		var descendantResp = response.GetDescendantRelevanceResponse{
			ServiceName:    descendant.Service,
			EndPoint:       descendant.Endpoint,
			Distance:       descendant.Relevance,
			DistanceType:   sortType,
			DelaySource:    "self",
			AlertStatus:    model.NORMAL_ALERT_STATUS,
			AlertReason:    model.AlertReason{},
			LastUpdateTime: nil,
		}

		// 填充延时占比信息 (DelaySource)
		fillServiceDelaySource(&descendantResp, descendantStatus)

		// 获取每个endpoint下的所有实例
		instances, err := s.promRepo.GetInstanceList(req.StartTime, req.EndTime, descendant.Service, descendant.Endpoint)
		if err != nil {
			// TODO deal error
			continue
		}

		startTime := time.UnixMicro(req.StartTime)
		endTime := time.UnixMicro(req.EndTime)

		instanceList := instances.GetInstances()

		// 填充Clickhouse侧的告警状态
		descendantResp.AlertStatusCH = serviceoverview.GetAlertStatusCH(
			s.chRepo, &descendantResp.AlertReason, []string{},
			descendant.Service, instanceList,
			startTime, endTime,
		)

		// 查询并填充进程启动时间
		startTSmap, _ := s.promRepo.QueryProcessStartTime(startTime, endTime, instanceList)
		latestStartTime := getLatestStartTime(startTSmap) * 1e6
		if latestStartTime > 0 {
			descendantResp.LastUpdateTime = &latestStartTime
		}
		resp = append(resp, descendantResp)
	}

	return resp, nil
}

func (s *service) queryDescendantDelaySource(services []string, endpoints []string, startTime, endTime int64) (map[string]*DelaySourceStatus, error) {
	avgLatency, err := s.promRepo.QueryAggMetricsWithFilter(
		prom.PQLAvgLatencyWithFilters,
		startTime, endTime,
		prom.EndpointGranularity,
		prom.ServiceRegexPQLFilter, prom.RegexMultipleValue(services...),
		prom.ContentKeyRegexPQLFilter, prom.RegexMultipleValue(endpoints...))
	if err != nil {
		return nil, err
	}

	avgDepLatency, err := s.promRepo.QueryAggMetricsWithFilter(
		prom.PQLAvgDepLatencyWithFilters,
		startTime, endTime,
		prom.EndpointGranularity,
		prom.ServiceRegexPQLFilter, prom.RegexMultipleValue(services...),
		prom.ContentKeyRegexPQLFilter, prom.RegexMultipleValue(endpoints...))
	if err != nil {
		return nil, err
	}

	var descendantStatusMap = make(map[string]*DelaySourceStatus)
	for _, metric := range avgLatency {
		status := &DelaySourceStatus{
			DepLatency: -1,
			Latency:    metric.Values[0].Value,
		}
		descendantStatusMap[metric.Metric.SvcName+"_"+metric.Metric.ContentKey] = status
	}

	for _, metric := range avgDepLatency {
		status, find := descendantStatusMap[metric.Metric.SvcName+"_"+metric.Metric.ContentKey]
		if find {
			status.DepLatency = metric.Values[0].Value
		}
	}

	return descendantStatusMap, err
}

type DelaySourceStatus struct {
	DepLatency float64
	Latency    float64
}

func fillServiceDelaySource(descendantResp *response.GetDescendantRelevanceResponse, descendantStatus map[string]*DelaySourceStatus) {
	ts := time.Now()
	descendantKey := descendantResp.ServiceName + "_" + descendantResp.EndPoint
	if status, ok := descendantStatus[descendantKey]; ok {
		if status.DepLatency > 0 && status.Latency > 0 {
			var depRatio = status.DepLatency / status.Latency
			if depRatio > 0.5 {
				descendantResp.DelaySource = "dependency"
			} else {
				descendantResp.DelaySource = "self"
			}
			delayDistribution := fmt.Sprintf("总延时: %.2f, 外部依赖延时: %.2f(%.2f)", status.DepLatency, status.Latency, depRatio)
			descendantResp.AlertReason.Add(model.DelaySourceAlert, model.AlertDetail{
				Timestamp:    ts.UnixMicro(),
				AlertObject:  descendantResp.ServiceName,
				AlertReason:  "外部依赖延时占总延时超过50%",
				AlertMessage: delayDistribution,
			})
		} else {
			descendantResp.DelaySource = "self"
		}
	} else {
		descendantResp.AlertReason.Add(model.DelaySourceAlert, model.AlertDetail{
			Timestamp:    ts.UnixMicro(),
			AlertObject:  descendantResp.ServiceName,
			AlertReason:  "时间段内未统计到请求,无法计算外部依赖延时占比",
			AlertMessage: "",
		})
	}
}

func getLatestStartTime(startTSmap map[model.ServiceInstance]int64) int64 {
	var latestStartTime int64 = -1
	for _, startTime := range startTSmap {
		if startTime > latestStartTime {
			latestStartTime = startTime
		}
	}
	return latestStartTime
}
