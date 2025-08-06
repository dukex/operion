// Package file provides file-based persistence implementation for workflows and triggers.
package file

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
)

type FilePersistence struct {
	root string
}

func NewFilePersistence(root string) persistence.Persistence {
	return &FilePersistence{
		root: strings.Replace(root, "file://", "", 1),
	}
}

// Close performs any necessary cleanup. For file-based persistence, there is nothing to clean up.
func (fp *FilePersistence) Close(_ context.Context) error {
	return nil
}

// HealthCheck checks if the file persistence layer is healthy by verifying the root directory exists.
func (fp *FilePersistence) HealthCheck(_ context.Context) error {
	if _, err := os.Stat(fp.root); os.IsNotExist(err) {
		return os.ErrNotExist
	}

	return nil
}

// Workflows retrieves all workflows from the file system.
func (fp *FilePersistence) Workflows(ctx context.Context) ([]*models.Workflow, error) {
	root := os.DirFS(fp.root + "/workflows")

	jsonFiles, err := fs.Glob(root, "*.json")
	if err != nil {
		return nil, err
	}

	if len(jsonFiles) == 0 {
		return make([]*models.Workflow, 0), nil
	}

	workflows := make([]*models.Workflow, 0, len(jsonFiles))

	for _, file := range jsonFiles {
		workflow, err := fp.WorkflowByID(ctx, file[:len(file)-5])
		if err != nil {
			return nil, err
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// WorkflowByID retrieves a workflow by its ID from the file system.
func (fp *FilePersistence) WorkflowByID(_ context.Context, workflowID string) (*models.Workflow, error) {
	filePath := filepath.Clean(path.Join(fp.root, "workflows", workflowID+".json"))

	body, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var workflow models.Workflow

	err = json.Unmarshal(body, &workflow)
	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

// SaveWorkflow saves a workflow to the file system.
func (fp *FilePersistence) SaveWorkflow(_ context.Context, workflow *models.Workflow) error {
	err := os.MkdirAll(fp.root+"/workflows", 0750)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}

	workflow.UpdatedAt = now

	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return err
	}

	filePath := path.Join(fp.root+"/workflows", workflow.ID+".json")

	return os.WriteFile(filePath, data, 0600)
}

// DeleteWorkflow removes a workflow by its ID.
func (fp *FilePersistence) DeleteWorkflow(_ context.Context, id string) error {
	filePath := path.Join(fp.root+"/workflows", id+".json")

	err := os.Remove(filePath)

	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}

// Schedule operations

func (fp *FilePersistence) Schedules() ([]*models.Schedule, error) {
	root := os.DirFS(fp.root + "/schedules")

	jsonFiles, err := fs.Glob(root, "*.json")
	if err != nil {
		return nil, err
	}

	if len(jsonFiles) == 0 {
		return make([]*models.Schedule, 0), nil
	}

	schedules := make([]*models.Schedule, 0, len(jsonFiles))

	for _, file := range jsonFiles {
		schedule, err := fp.ScheduleByID(file[:len(file)-5])
		if err != nil {
			return nil, err
		}

		if schedule != nil {
			schedules = append(schedules, schedule)
		}
	}

	return schedules, nil
}

func (fp *FilePersistence) ScheduleByID(scheduleID string) (*models.Schedule, error) {
	filePath := path.Join(fp.root+"/schedules", scheduleID+".json")

	// Validate that the resolved path is within the expected directory to prevent path traversal
	if !strings.HasPrefix(filePath, fp.root+"/schedules/") {
		return nil, errors.New("invalid schedule ID: path traversal detected")
	}

	body, err := os.ReadFile(filePath) // #nosec G304 -- path is validated above
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var schedule models.Schedule

	err = json.Unmarshal(body, &schedule)
	if err != nil {
		return nil, err
	}

	return &schedule, nil
}

func (fp *FilePersistence) ScheduleBySourceID(sourceID string) (*models.Schedule, error) {
	schedules, err := fp.Schedules()
	if err != nil {
		return nil, err
	}

	for _, schedule := range schedules {
		if schedule.SourceID == sourceID {
			return schedule, nil
		}
	}

	return nil, nil
}

func (fp *FilePersistence) SaveSchedule(schedule *models.Schedule) error {
	err := os.MkdirAll(fp.root+"/schedules", 0750)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}

	schedule.UpdatedAt = now

	data, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return err
	}

	filePath := path.Join(fp.root+"/schedules", schedule.ID+".json")

	return os.WriteFile(filePath, data, 0600)
}

func (fp *FilePersistence) DeleteSchedule(id string) error {
	filePath := path.Join(fp.root+"/schedules", id+".json")

	err := os.Remove(filePath)

	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}

func (fp *FilePersistence) DeleteScheduleBySourceID(sourceID string) error {
	schedule, err := fp.ScheduleBySourceID(sourceID)
	if err != nil {
		return err
	}

	if schedule == nil {
		return nil
	}

	return fp.DeleteSchedule(schedule.ID)
}

func (fp *FilePersistence) DueSchedules(before time.Time) ([]*models.Schedule, error) {
	schedules, err := fp.Schedules()
	if err != nil {
		return nil, err
	}

	dueSchedules := make([]*models.Schedule, 0)

	for _, schedule := range schedules {
		if schedule.IsDue(before) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}

	return dueSchedules, nil
}

// WorkflowTriggersBySourceID returns workflow triggers that match a specific source ID and workflow status.
func (fp *FilePersistence) WorkflowTriggersBySourceID(ctx context.Context, sourceID string, status models.WorkflowStatus) ([]*models.TriggerMatch, error) {
	workflows, err := fp.Workflows(ctx)
	if err != nil {
		return nil, err
	}

	var matchingTriggers []*models.TriggerMatch

	for _, wf := range workflows {
		// Only process workflows with the specified status
		if wf.Status != status {
			continue
		}

		for _, trigger := range wf.WorkflowTriggers {
			// Check if this trigger matches the source ID
			if trigger.SourceID == sourceID {
				matchingTriggers = append(matchingTriggers, &models.TriggerMatch{
					WorkflowID: wf.ID,
					Trigger:    trigger,
				})
			}
		}
	}

	return matchingTriggers, nil
}

// WorkflowTriggersBySourceAndEvent returns workflow triggers that match a specific source ID, event type, and workflow status.
func (fp *FilePersistence) WorkflowTriggersBySourceAndEvent(ctx context.Context, sourceID, eventType string, status models.WorkflowStatus) ([]*models.TriggerMatch, error) {
	workflows, err := fp.Workflows(ctx)
	if err != nil {
		return nil, err
	}

	var matchingTriggers []*models.TriggerMatch

	for _, wf := range workflows {
		// Only process workflows with the specified status
		if wf.Status != status {
			continue
		}

		for _, trigger := range wf.WorkflowTriggers {
			// Check if this trigger matches the source ID
			if trigger.SourceID != sourceID {
				continue
			}

			// TODO: Add event type filtering based on trigger configuration
			// For now, any event from the matching source will trigger the workflow
			// Future enhancement: triggers could specify which event types they're interested in
			// via their Configuration map, e.g., trigger.Configuration["event_types"] = ["ScheduleDue", "ScheduleOverdue"]

			matchingTriggers = append(matchingTriggers, &models.TriggerMatch{
				WorkflowID: wf.ID,
				Trigger:    trigger,
			})
		}
	}

	return matchingTriggers, nil
}
