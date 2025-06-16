package domain

import (
	log "github.com/sirupsen/logrus"
)

type ExecutionContext struct {
	WorkflowID  string
	ExecutionID string
	TriggerData map[string]interface{}
	Variables   map[string]interface{}
	StepResults map[string]interface{}
	Metadata    map[string]interface{}
	Logger      *log.Entry
}

func (ex *ExecutionContext) WithLogger(logger *log.Entry) *ExecutionContext {
	return &ExecutionContext{
		WorkflowID:  ex.WorkflowID,
		ExecutionID: ex.ExecutionID,
		TriggerData: ex.TriggerData,
		Variables:   ex.Variables,
		StepResults: ex.StepResults,
		Metadata:    ex.Metadata,
		Logger:      logger,
	}
}
