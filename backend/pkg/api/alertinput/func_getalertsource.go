// Copyright 2024 CloudDetail
// SPDX-License-Identifier: Apache-2.0

package alert

import (
	"net/http"

	"github.com/CloudDetail/apo/backend/pkg/code"
	"github.com/CloudDetail/apo/backend/pkg/core"
	"github.com/CloudDetail/apo/backend/pkg/model/integration/alert"
)

// GetAlertSource GetAlertSource
// @Summary GetAlertSource
// @Description GetAlertSource
// @Tags API.alertinput
// @Accept application/json
// @Produce json
// @Param Request body alert.SourceFrom true "Source Info"
// @Success 200 {object} alert.AlertSource
// @Failure 400 {object} code.Failure
// @Router /api/alertinput/source/get [post]
func (h *handler) GetAlertSource() core.HandlerFunc {
	return func(c core.Context) {
		req := new(alert.SourceFrom)
		if err := c.ShouldBindJSON(req); err != nil {
			c.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.ParamBindError,
				c.ErrMessage(code.ParamBindError)).WithError(err),
			)
			return
		}

		resp, err := h.inputService.GetAlertSource(req)
		if err != nil {
			c.AbortWithError(core.Error(
				http.StatusBadRequest,
				code.CreateAlertSourceFailed,
				// TODO ErrorCode
				c.ErrMessage(code.GetAlertSourceFailed)).WithError(err),
			)
			return
		}

		c.Payload(resp)
	}
}
