// Copyright 2024 CloudDetail
// SPDX-License-Identifier: Apache-2.0

package request

import "github.com/CloudDetail/apo/backend/pkg/model"

type CreateDataGroupRequest struct {
	GroupName      string             `json:"groupName" binding:"required"`
	Description    string             `json:"description"`
	DatasourceList []model.Datasource `json:"datasourceList"`
	// AssignedSubjects []AuthDataGroup    `json:"assignedSubjects"`
}

type AuthDataGroup struct {
	SubjectID int64  `json:"subjectId"`
	Type      string `json:"type"` // edit or view
}

type DeleteDataGroupRequest struct {
	GroupID int64 `form:"groupId" binding:"required"`
}

type UpdateDataGroupRequest struct {
	GroupID        int64              `json:"groupId" form:"groupId" binding:"required"`
	GroupName      string             `json:"groupName" form:"groupName" binding:"required"`
	Description    string             `json:"description"`
	DatasourceList []model.Datasource `json:"datasourceList"`
}

type GetDataGroupRequest struct {
	GroupName      string             `json:"groupName" form:"groupName"`
	DataSourceList []model.Datasource `json:"datasourceList"`
	*PageParam
}

type GetGroupDatasourceRequest struct {
	GroupID  int64  `form:"groupId"`
	Category string `form:"category"` // apm or normal
}

type GetSubjectDataGroupRequest struct {
	SubjectID   int64  `form:"subjectId" binding:"required"`
	SubjectType string `form:"subjectType" binding:"required,oneof=user team"`
	Category    string `form:"category"`
}

type GetUserDataGroupRequest struct {
	UserID   int64  `form:"userId" binding:"required"`
	Category string `form:"category"`
}

type GroupSubsOperationRequest struct {
	DataGroupID int64           `json:"groupId" form:"groupId" binding:"required"`
	UserList    []AuthDataGroup `json:"userList"`
	TeamList    []AuthDataGroup `json:"teamList"`
}

type GetGroupSubsRequest struct {
	DataGroupID int64  `form:"groupId" binding:"required"`
	SubjectType string `form:"subjectType"`
}

