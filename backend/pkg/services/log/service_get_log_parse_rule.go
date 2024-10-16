package log

import (
	"regexp"
	"strings"

	"github.com/CloudDetail/apo/backend/pkg/model/request"
	"github.com/CloudDetail/apo/backend/pkg/model/response"
	"github.com/CloudDetail/apo/backend/pkg/repository/database"
)

var routeReg = regexp.MustCompile(`\"(.*?)\"`)

func getRouteRuleMap(routeRule string) map[string]string {
	res := make(map[string][]string)
	lines := strings.Split(routeRule, "||")
	for _, line := range lines {
		if line == "" {
			continue
		}
		matches := routeReg.FindAllStringSubmatch(line, -1)
		if len(matches) == 2 {
			key := matches[0][1]
			value := matches[1][1]
			res[key] = append(res[key], value)
		}
	}
	rc := make(map[string]string)
	for k, v := range res {
		rc[k] = strings.Join(v, ",")
	}
	return rc
}

func (s *service) GetLogParseRule(req *request.QueryLogParseRequest) (*response.LogParseResponse, error) {
	model := &database.LogTableInfo{
		DataBase: req.DataBase,
		Table:    req.TableName,
	}
	err := s.dbRepo.OperateLogTableInfo(model, database.QUERY)
	if err != nil {
		return &response.LogParseResponse{
			ParseName: defaultParseName,
			ParseRule: defaultParseRule,
			RouteRule: defaultRouteRuleMap,
		}, err
	}

	return &response.LogParseResponse{
		Service:   model.Service,
		ParseName: model.ParseName,
		ParseRule: model.ParseRule,
		RouteRule: getRouteRuleMap(model.RouteRule),
	}, nil
}
