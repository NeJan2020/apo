// Copyright 2025 CloudDetail
// SPDX-License-Identifier: Apache-2.0

package clickhouse

import (
	"context"
	"time"

	"github.com/CloudDetail/apo/backend/pkg/model"
)

func (ch *chRepo) AddWorkflowRecords(ctx context.Context, records []model.WorkflowRecord) error {
	batch, err := ch.conn.PrepareBatch(ctx, `
		INSERT INTO workflow_records (workflow_run_id, workflow_id, workflow_name, ref, input, output, created_at)
		VALUES
	`)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := batch.Append(
			record.WorkflowRunID,
			record.WorkflowID,
			record.WorkflowName,
			record.Ref,
			record.Input,
			record.Output,
			time.UnixMicro(record.CreatedAt),
		); err != nil {
			continue
		}
	}

	if err := batch.Send(); err != nil {
		return err
	}
	return nil
}
