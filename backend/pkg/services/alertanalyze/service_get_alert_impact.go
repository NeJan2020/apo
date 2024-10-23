package alertanalyze

import (
	"errors"
	"fmt"
	"time"

	"github.com/CloudDetail/apo/backend/pkg/model"
	"github.com/CloudDetail/apo/backend/pkg/repository/clickhouse"
	"github.com/CloudDetail/apo/backend/pkg/repository/prometheus"
)

// AlertImpact 分析告警事件的影响面
// 1. 根据告警时间类型找到关联的Service,
// !!! 会检查Event中是否有满足要求的Label,如果没有会尝试所有预设的label组合
// 2. 通过ServiceTopology查询service的关联入口
func (s *service) AlertImpact(eventid string, startTimeTs, endTimeTs int64) ([]clickhouse.EntryNode, error) {
	startTime := time.UnixMicro(startTimeTs)
	endTime := time.UnixMicro(endTimeTs)

	// 从Clickhouse中获取到对应的告警
	// if eventid == "" {
	// 	events, count, err := s.chRepo.GetAlertEvents(startTime, endTime, request.AlertFilter{}, nil, nil, clickhouse.ReceivedTimeOrders)

	// }
	event, err := s.chRepo.GetAlertEventById(eventid, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert eventId[%s]: %v", eventid, err)
	}

	// 根据告警事件类型查询影响入口
	var endpoints []clickhouse.AlertService
	switch clickhouse.AlertGroup(event.Group) {
	case clickhouse.APP_GROUP:
		endpoints, err = s.tryGetAlertServiceByService(event, startTime, endTime)
	case clickhouse.NETWORK_GROUP:
		// 先尝试K8sPod匹配
		endpoints, err = s.tryGetAlertServiceByContainer(event, startTime, endTime)
		if err != nil && errors.As(err, &model.ErrAlertImpactMissingTag{}) {
			// 再尝试Node匹配
			endpoints, err = s.tryGetAlertServiceByNetSrcVM(event, startTime, endTime)
		}
	case clickhouse.CONTAINER_GROUP:
		endpoints, err = s.tryGetAlertServiceByContainer(event, startTime, endTime)
	case clickhouse.INFRA_GROUP:
		endpoints, err = s.tryGetAlertServiceByInfraNode(event, startTime, endTime)
	}

	if err != nil && errors.As(err, &model.ErrAlertImpactMissingTag{}) {
		// 预期的Label不存在,尝试所有预设的label组合
		endpoints, err = s.tryGetAlertService(event, startTime, endTime)
	}

	if err != nil {
		return nil, err
	}

	// 通过ServiceTopology关联查询入口
	return s.chRepo.SearchEntryEndpointsByAlertService(endpoints, startTime.UnixMicro(), endTime.UnixMicro())
}

func (s *service) tryGetAlertService(event *model.AlertEvent, startTime time.Time, endTime time.Time) ([]clickhouse.AlertService, error) {
	var tryMethods = []func(*model.AlertEvent, time.Time, time.Time) ([]clickhouse.AlertService, error){
		s.tryGetAlertServiceByService,
		s.tryGetAlertServiceByContainer,
		s.tryGetAlertServiceByNetSrcVM,
		s.tryGetAlertServiceByInfraNode,
	}
	var endpoints []clickhouse.AlertService
	checkedError := model.ErrAlertImpactMissingTag{
		TagGroups: []model.TagGroup{},
		Event:     event,
	}
	for _, tryGetService := range tryMethods {
		var err error
		endpoints, err = tryGetService(event, startTime, endTime)
		if err == nil {
			break
		}
		// 如果是Tag不足,继续尝试别的Tag
		var vErr model.ErrAlertImpactMissingTag
		if errors.As(err, &vErr) {
			checkedError.AddCheckedGroup(vErr)
			continue
		}
		// 其他错误,直接返回
		return nil, err
	}

	return endpoints, checkedError
}

func (s *service) tryGetAlertServiceByContainer(event *model.AlertEvent, startTime time.Time, endTime time.Time) ([]clickhouse.AlertService, error) {
	podName := event.GetK8sPodTag()
	namespace := event.GetK8sNamespaceTag()
	if len(podName) == 0 || len(namespace) == 0 {
		return nil, model.ErrAlertImpactMissingTag{
			TagGroups: []model.TagGroup{[]string{"pod", "namespace"}},
			Event:     event,
		}
	}

	// 通常也只会有一个Service
	services, err := s.promRepo.GetServiceListByFilter(
		startTime, endTime,
		prometheus.NamespacePQLFilter, event.GetK8sNamespaceTag(),
		prometheus.PodPQLFilter, event.GetK8sPodTag(),
	)
	if err != nil {
		return nil, err
	}
	var endpoints []clickhouse.AlertService
	// 通常只有一个service
	for _, service := range services {
		// 不关系ContentKey
		endpoints = append(endpoints, clickhouse.AlertService{
			ServiceName: service,
		})
	}

	if len(endpoints) == 0 {
		return nil, model.ErrAlertImpactNoMatchedService{
			TagGroup:  []string{"pod", "namespace"},
			TagValues: []string{podName, namespace},
		}
	}
	return endpoints, nil
}

func (s *service) tryGetAlertServiceByService(event *model.AlertEvent, _ time.Time, _ time.Time) ([]clickhouse.AlertService, error) {
	serviceName := event.GetServiceNameTag()
	if len(serviceName) == 0 {
		return nil, model.ErrAlertImpactMissingTag{
			TagGroups: []model.TagGroup{[]string{"svc_name"}},
			Event:     event,
		}
	}

	return []clickhouse.AlertService{
		{
			ServiceName: serviceName,
			ContentKey:  event.GetContentKeyTag(),
		},
	}, nil
}

func (s *service) tryGetAlertServiceByNetSrcVM(event *model.AlertEvent, startTime time.Time, endTime time.Time) ([]clickhouse.AlertService, error) {
	nodeName := event.GetNetSrcNodeTag()
	pid := event.GetNetSrcPidTag()
	if len(nodeName) == 0 || len(pid) == 0 {
		return nil, model.ErrAlertImpactMissingTag{
			TagGroups: []model.TagGroup{[]string{"node", "pid"}},
			Event:     event,
		}
	}

	services, err := s.promRepo.GetServiceListByFilter(
		startTime, endTime,
		prometheus.NodePQLFilter, nodeName,
		prometheus.PidPQLFilter, pid,
	)

	if err != nil {
		return nil, err
	}

	var endpoints []clickhouse.AlertService
	for _, service := range services {
		endpoints = append(endpoints, clickhouse.AlertService{
			ServiceName: service,
		})
	}

	if len(endpoints) == 0 {
		return nil, model.ErrAlertImpactNoMatchedService{
			TagGroup:  []string{"node_name", "pid"},
			TagValues: []string{nodeName, pid},
		}
	}

	return endpoints, nil
}

func (s *service) tryGetAlertServiceByInfraNode(event *model.AlertEvent, startTime time.Time, endTime time.Time) ([]clickhouse.AlertService, error) {
	nodeName := event.GetInfraNodeTag()
	if len(nodeName) == 0 {
		return nil, model.ErrAlertImpactMissingTag{
			TagGroups: []model.TagGroup{[]string{"instance_name"}},
			Event:     event,
		}
	}

	services, err := s.promRepo.GetServiceListByFilter(
		startTime, endTime,
		prometheus.NodePQLFilter, nodeName,
	)

	if err != nil {
		return nil, err
	}

	var endpoints []clickhouse.AlertService
	for _, service := range services {
		endpoints = append(endpoints, clickhouse.AlertService{
			ServiceName: service,
		})
	}

	if len(endpoints) == 0 {
		return nil, model.ErrAlertImpactNoMatchedService{
			TagGroup:  []string{"node"},
			TagValues: []string{nodeName},
		}
	}

	return endpoints, nil
}
