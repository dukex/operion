package models

import (
	log "github.com/sirupsen/logrus"
)

type ExecutionContext struct {
	ID          string
	WorkflowID  string
	TriggerData map[string]interface{}
	Variables   map[string]interface{}
	StepResults map[string]interface{}
	Metadata    map[string]interface{}
	Logger      *log.Entry
}

func (ex *ExecutionContext) WithLogger(logger *log.Entry) *ExecutionContext {
	return &ExecutionContext{
		WorkflowID:  ex.WorkflowID,
		ID:          ex.ID,
		TriggerData: ex.TriggerData,
		Variables:   ex.Variables,
		StepResults: ex.StepResults,
		Metadata:    ex.Metadata,
		Logger:      logger,
	}
}
