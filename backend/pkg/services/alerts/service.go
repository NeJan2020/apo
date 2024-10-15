package alerts

import (
	"github.com/CloudDetail/apo/backend/pkg/model/request"
	"github.com/CloudDetail/apo/backend/pkg/model/response"
	"github.com/CloudDetail/apo/backend/pkg/repository/clickhouse"
	"github.com/CloudDetail/apo/backend/pkg/repository/kubernetes"
	"github.com/CloudDetail/apo/backend/pkg/repository/prometheus"
)

var _ Service = (*service)(nil)

type Service interface {
	// ========================告警分析========================

	// AlertImpact 获取告警事件的影响面
	// 如果关联所需的Label不足,error会返回ErrAlertImpactMissingTag提示期望哪些tag
	AlertImpact(eventid string, startTime, endTime int64) ([]clickhouse.EntryNode, error)

	// ========================告警配置========================

	// InputAlertManager 接收 AlertManager 的告警事件
	InputAlertManager(req *request.InputAlertManagerRequest) error

	// GetAlertRuleFile 获取基础告警规则
	GetAlertRuleFile(req *request.GetAlertRuleConfigRequest) (*response.GetAlertRuleFileResponse, error)
	// UpdateAlertRuleFile 更新告警基础规则
	UpdateAlertRuleFile(req *request.UpdateAlertRuleConfigRequest) error

	// AlertRule Options
	GetGroupList() response.GetGroupListResponse
	GetMetricPQL() response.GetMetricPQLResponse

	// AlertRule CRUD
	GetAlertRules(req *request.GetAlertRuleRequest) response.GetAlertRulesResponse
	UpdateAlertRule(req *request.UpdateAlertRuleRequest) error
	DeleteAlertRule(req *request.DeleteAlertRuleRequest) error
	AddAlertRule(req *request.AddAlertRuleRequest) error
	CheckAlertRule(req *request.CheckAlertRuleRequest) (response.CheckAlertRuleResponse, error)

	// AlertManager Receiver CRUD
	GetAMConfigReceivers(req *request.GetAlertManagerConfigReceverRequest) response.GetAlertManagerConfigReceiverResponse
	AddAMConfigReceiver(req *request.AddAlertManagerConfigReceiver) error
	UpdateAMConfigReceiver(req *request.UpdateAlertManagerConfigReceiver) error
	DeleteAMConfigReceiver(req *request.DeleteAlertManagerConfigReceiverRequest) error
}

type service struct {
	chRepo   clickhouse.Repo
	promRepo prometheus.Repo
	k8sApi   kubernetes.Repo
}

func New(chRepo clickhouse.Repo, promRepo prometheus.Repo, k8sApi kubernetes.Repo) Service {
	return &service{
		chRepo:   chRepo,
		promRepo: promRepo,
		k8sApi:   k8sApi,
	}
}
