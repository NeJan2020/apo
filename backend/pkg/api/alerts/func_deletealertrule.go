package alerts

import (
	"net/http"

	"github.com/CloudDetail/apo/backend/pkg/code"
	"github.com/CloudDetail/apo/backend/pkg/core"
	"github.com/CloudDetail/apo/backend/pkg/model/request"
)

// DeleteAlertRule 删除告警规则
// @Summary 删除告警规则
// @Description 删除告警规则
// @Tags API.alerts
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param alertRuleFile query string false "告警规则文件名"
// @Param group query string false "告警规则组"
// @Param alert query string false "告警规则名"
// @Success 200 string ok
// @Failure 400 {object} code.Failure
// @Router /api/alerts/rule [delete]
func (h *handler) DeleteAlertRule() core.HandlerFunc {
	return func(c core.Context) {
		req := new(request.DeleteAlertRuleRequest)
		if err := c.ShouldBindQuery(req); err != nil {
			c.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				code.Text(code.ParamBindError)).WithError(err),
			)
			return
		}

		err := h.alertService.DeleteAlertRule(req)
		if err != nil {
			c.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.DeleteAlertRuleError,
				code.Text(code.DeleteAlertRuleError)).WithError(err),
			)
			return
		}

		c.Payload("ok")
	}
}
