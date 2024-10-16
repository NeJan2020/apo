package alerts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CloudDetail/apo/backend/pkg/model"
	"github.com/CloudDetail/apo/backend/pkg/model/request"
	"github.com/CloudDetail/apo/backend/pkg/model/response"
	ck "github.com/CloudDetail/apo/backend/pkg/repository/clickhouse"
)

func (s *service) SearchAnormalEventByEntry(req *request.GetDescendantAnormalEventRequest) (*response.GetDescendantAnormalEventResponse, error) {
	startTime := time.UnixMicro(req.StartTime)
	endTime := time.UnixMicro(req.EndTime)

	descendants, err := s.chRepo.ListDescendantNodes(&request.GetDescendantMetricsRequest{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Service:   req.Service,
		Endpoint:  req.Endpoint,
	})
	if err != nil {
		return nil, err
	}

	selectedEvents := strings.Split(req.SelectedEventType, ",")

	// 便于后续查询受告警影响的服务
	var instances []*model.ServiceInstance
	var endpoints []model.EndpointKey
	instanceMap := newInstanceMap()
	for _, descendant := range descendants {
		// 获取每个endpoint下的所有实例
		instanceList, err := s.promRepo.GetInstanceList(req.StartTime, req.EndTime, descendant.Service, descendant.Endpoint)
		if err != nil {
			continue
		}

		// 构建好子孙节点的Node/Service -> descendant 映射
		endpoint := model.EndpointKey{
			ServiceName: descendant.Service,
			ContentKey:  descendant.Endpoint,
		}
		instancesForDescendant := instanceList.GetInstances()
		instanceMap.AddInstances(endpoint, instancesForDescendant)

		instances = append(instances, instancesForDescendant...)
		endpoints = append(endpoints, endpoint)
	}

	// 返回结果列表
	var anormalEventList []model.AnormalEvent

	// 获取匹配的error
	if len(selectedEvents) == 0 || contains(selectedEvents, "error") {
		propagations, err := s.chRepo.ListErrorByEntryService(req.StartTime, req.EndTime, req.Service, req.Endpoint, endpoints)
		if err == nil {
			errorEvents := s.parseErrorEvent(propagations, instanceMap)
			// 存放error事件
			anormalEventList = append(anormalEventList, errorEvents...)
		}

	}

	if len(selectedEvents) == 0 || contains(selectedEvents, "alert") {
		// 获取匹配的alertEvents
		alertEvents, _, err := s.chRepo.GetAlertEventsByInstanceAndEndpoints(
			startTime, endTime,
			request.AlertFilter{Status: "firing"},
			instances, endpoints, nil, ck.OrderAlertByReceivedTime,
		)

		if err == nil {
			anormalAlerts := s.parseAlertEvents(alertEvents, instanceMap)
			// 存放告警事件
			anormalEventList = append(anormalEventList, anormalAlerts...)
		}
	}

	// TODO 获取Log错误告警事件/K8s事件等

	sort.SliceStable(anormalEventList, func(i, j int) bool {
		return anormalEventList[i].Timestamp < anormalEventList[j].Timestamp
	})

	return &response.GetDescendantAnormalEventResponse{
		AnormalEvents: anormalEventList,
	}, nil
}

func (*service) parseErrorEvent(propagations []ck.ErrorPropation, instanceMap *InstanceMap) []model.AnormalEvent {
	var anormalEventList []model.AnormalEvent
	for _, propagation := range propagations {
		errorEvent := model.AnormalEvent{
			Timestamp:   propagation.Timestamp.UnixMicro(),
			AnormalType: model.AnormalTypeError,
		}
		for idx, service := range propagation.NodesService {
			if !propagation.NodesIsError[idx] {
				continue
			}
			// 检查数据存在,防止数组越界
			if len(propagation.NodesUrl) <= idx ||
				len(propagation.NodesInstance) <= idx ||
				len(propagation.NodesErrorTypes) <= idx ||
				len(propagation.NodesErrorMsgs) <= idx {
				break
			}
			endpointKey := model.EndpointKey{
				ServiceName: service,
				ContentKey:  propagation.NodesUrl[idx],
			}
			// 检查ContentKey是否在endpoints中
			if !instanceMap.IsEndpointKeyExist(endpointKey) {
				continue
			}
			errorEvent.ImpactEndpoints = append(errorEvent.ImpactEndpoints, model.AnormalEventDetail{
				EndpointKey:  endpointKey,
				AlertObject:  propagation.NodesInstance[idx],
				AlertReason:  strings.Join(propagation.NodesErrorTypes[idx], ";"),
				AlertMessage: strings.Join(propagation.NodesErrorMsgs[idx], ";"),
			})
		}
		anormalEventList = append(anormalEventList, errorEvent)
	}
	return anormalEventList
}

func (*service) parseAlertEvents(alertEvents []ck.PagedAlertEvent, instanceMap *InstanceMap) []model.AnormalEvent {
	var anormalEventList []model.AnormalEvent
	for _, alertEvent := range alertEvents {
		var anormalEvent model.AnormalEvent = model.AnormalEvent{
			Timestamp:       alertEvent.ReceivedTime.UnixMicro(),
			AnormalType:     0,
			ImpactEndpoints: []model.AnormalEventDetail{},
		}
		switch ck.AlertGroup(alertEvent.Group) {
		case ck.APP_GROUP:
			anormalEvent.AnormalType = model.AnormalTypeAlertApp
			anormalEvent.ImpactEndpoints = append(anormalEvent.ImpactEndpoints, model.AnormalEventDetail{
				EndpointKey: model.EndpointKey{
					ServiceName: alertEvent.GetServiceNameTag(),
					ContentKey:  alertEvent.GetContentKeyTag(),
				},
				AlertObject:  alertEvent.GetTargetObj(),
				AlertReason:  alertEvent.Name,
				AlertMessage: alertEvent.Detail,
			})
		case ck.CONTAINER_GROUP:
			anormalEvent.AnormalType = model.AnormalTypeAlertContainer
			instance, endpoints := instanceMap.GetEndpointsByK8sPodNS(alertEvent.GetK8sPodTag(), alertEvent.GetK8sNamespaceTag())
			if instance == nil {
				continue
			}
			for _, endpoint := range endpoints {
				anormalEvent.ImpactEndpoints = append(anormalEvent.ImpactEndpoints, model.AnormalEventDetail{
					EndpointKey:  endpoint,
					AlertObject:  alertEvent.GetTargetObj(),
					AlertReason:  alertEvent.Name,
					AlertMessage: alertEvent.Detail,
				})
			}
		case ck.NETWORK_GROUP:
			anormalEvent.AnormalType = model.AnormalTypeAlertNet
			var endpoints []model.EndpointKey
			var instance *model.ServiceInstance
			pod := alertEvent.GetK8sPodTag()
			if len(pod) > 0 {
				instance, endpoints = instanceMap.GetEndpointsByK8sPodNS(alertEvent.GetK8sPodTag(), alertEvent.GetK8sNamespaceTag())
				if instance == nil {
					continue
				}
			} else {
				instance, endpoints = instanceMap.GetEndpointsByNodePid(alertEvent.GetNetSrcNodeTag(), alertEvent.GetNetSrcPidTag())
				if instance == nil {
					continue
				}
			}
			for _, endpoint := range endpoints {

				anormalEvent.ImpactEndpoints = append(anormalEvent.ImpactEndpoints, model.AnormalEventDetail{
					EndpointKey:  endpoint,
					AlertObject:  alertEvent.GetTargetObj(),
					AlertReason:  alertEvent.Name,
					AlertMessage: alertEvent.Detail,
				})
			}
		case ck.INFRA_GROUP:
			anormalEvent.AnormalType = model.AnormalTypeAlertInfra
			endpointsMaps := instanceMap.GetEndpointsByNode(alertEvent.GetInfraNodeTag())
			for instance, endpoints := range endpointsMaps {
				var instanceName string
				if len(instance.PodName) > 0 {
					instanceName = fmt.Sprintf("%s/%s", instance.Namespace, instance.PodName)
				} else {
					instanceName = fmt.Sprintf("(pid:%d)", instance.Pid)
				}
				for _, endpoint := range endpoints {
					anormalEvent.ImpactEndpoints = append(anormalEvent.ImpactEndpoints, model.AnormalEventDetail{
						EndpointKey:  endpoint,
						AlertObject:  instanceName + " at " + alertEvent.GetTargetObj(),
						AlertReason:  alertEvent.Name,
						AlertMessage: alertEvent.Detail,
					})
				}
			}
		}
		anormalEventList = append(anormalEventList, anormalEvent)
	}
	return anormalEventList
}

type InstanceMap struct {
	Pod2InstanceMap     map[K8sPodNSKey]model.ServiceInstance
	NodePid2InstanceMap map[NodePidKey]model.ServiceInstance
	Node2InstancesMap   map[string]map[model.ServiceInstance]struct{}

	InstanceMap map[model.ServiceInstance]map[model.EndpointKey]struct{}

	EndpointMap map[model.EndpointKey]struct{}
}

type NodePidKey struct {
	Node string
	Pid  int
}

type K8sPodNSKey struct {
	Namespace string
	Pod       string
}

func newInstanceMap() *InstanceMap {
	return &InstanceMap{
		Pod2InstanceMap:     map[K8sPodNSKey]model.ServiceInstance{},
		NodePid2InstanceMap: map[NodePidKey]model.ServiceInstance{},
		Node2InstancesMap:   map[string]map[model.ServiceInstance]struct{}{},
		InstanceMap:         map[model.ServiceInstance]map[model.EndpointKey]struct{}{},
		EndpointMap:         map[model.EndpointKey]struct{}{},
	}
}

func (m *InstanceMap) AddInstances(endpointKey model.EndpointKey, instances []*model.ServiceInstance) {
	m.EndpointMap[endpointKey] = struct{}{}

	for _, instance := range instances {
		endpointKeys, find := m.InstanceMap[*instance]
		if !find {
			endpointKeys = make(map[model.EndpointKey]struct{})
		}
		endpointKeys[endpointKey] = struct{}{}
		m.InstanceMap[*instance] = endpointKeys

		if len(instance.PodName) > 0 {
			m.Pod2InstanceMap[K8sPodNSKey{instance.Namespace, instance.PodName}] = *instance
		}

		if instance.Pid > 0 {
			m.NodePid2InstanceMap[NodePidKey{instance.NodeName, int(instance.Pid)}] = *instance
		}

		if len(instance.NodeName) > 0 {
			instancesOnNode, find := m.Node2InstancesMap[instance.NodeName]
			if !find {
				instancesOnNode = make(map[model.ServiceInstance]struct{})
			}
			instancesOnNode[*instance] = struct{}{}
			m.Node2InstancesMap[instance.NodeName] = instancesOnNode
		}
	}
}

func (m *InstanceMap) GetEndpointsByK8sPodNS(pod, namespace string) (*model.ServiceInstance, []model.EndpointKey) {
	instance, find := m.Pod2InstanceMap[K8sPodNSKey{namespace, pod}]
	if !find {
		return nil, nil
	}

	endpointsMap, find := m.InstanceMap[instance]
	if !find {
		return nil, nil
	}
	var endpoints []model.EndpointKey
	for endpoint := range endpointsMap {
		endpoints = append(endpoints, endpoint)
	}
	return &instance, endpoints
}

func (m *InstanceMap) GetEndpointsByNodePid(node string, pid string) (*model.ServiceInstance, []model.EndpointKey) {
	if len(pid) == 0 {

	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		return nil, nil
	}

	instance, find := m.NodePid2InstanceMap[NodePidKey{node, pidInt}]
	if !find {
		return nil, nil
	}

	endpointsMap, find := m.InstanceMap[instance]
	if !find {
		return nil, nil
	}
	var endpoints []model.EndpointKey
	for endpoint := range endpointsMap {
		endpoints = append(endpoints, endpoint)
	}
	return &instance, endpoints
}

func (m *InstanceMap) GetEndpointsByNode(node string) map[model.ServiceInstance][]model.EndpointKey {
	instances, find := m.Node2InstancesMap[node]
	if !find || len(instances) == 0 {
		return nil
	}

	var res = make(map[model.ServiceInstance][]model.EndpointKey)
	for instance := range instances {
		endpointsMap, find := m.InstanceMap[instance]
		if !find {
			continue
		}
		var endpoints []model.EndpointKey
		for endpoint := range endpointsMap {
			endpoints = append(endpoints, endpoint)
		}

		res[instance] = endpoints
	}

	return res
}

func (m *InstanceMap) IsEndpointKeyExist(endpointKey model.EndpointKey) bool {
	_, find := m.EndpointMap[endpointKey]
	return find
}

func contains(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}
