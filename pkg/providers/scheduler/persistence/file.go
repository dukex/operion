package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/providers/scheduler/models"
)

// FilePersistence implements SchedulerPersistence using JSON files.
type FilePersistence struct {
	dataDir   string
	mu        sync.RWMutex
	schedules map[string]*models.Schedule
}

// NewFilePersistence creates a new file-based scheduler persistence.
func NewFilePersistence(dataDir string) (*FilePersistence, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fp := &FilePersistence{
		dataDir:   dataDir,
		schedules: make(map[string]*models.Schedule),
	}

	// Load existing schedules
	if err := fp.loadSchedules(); err != nil {
		return nil, fmt.Errorf("failed to load schedules: %w", err)
	}

	return fp, nil
}

// SaveSchedule saves a schedule to the file system.
func (fp *FilePersistence) SaveSchedule(schedule *models.Schedule) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.schedules[schedule.ID] = schedule

	return fp.saveSchedulesToFile()
}

// ScheduleByID retrieves a schedule by its ID.
func (fp *FilePersistence) ScheduleByID(id string) (*models.Schedule, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	schedule, exists := fp.schedules[id]
	if !exists {
		return nil, nil
	}

	return schedule, nil
}

// ScheduleBySourceID retrieves a schedule by its source ID.
func (fp *FilePersistence) ScheduleBySourceID(sourceID string) (*models.Schedule, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	for _, schedule := range fp.schedules {
		if schedule.SourceID == sourceID {
			return schedule, nil
		}
	}

	return nil, nil
}

// Schedules returns all schedules.
func (fp *FilePersistence) Schedules() ([]*models.Schedule, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	schedules := make([]*models.Schedule, 0, len(fp.schedules))
	for _, schedule := range fp.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// DueSchedules returns schedules that are due before the given time.
func (fp *FilePersistence) DueSchedules(before time.Time) ([]*models.Schedule, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	var dueSchedules []*models.Schedule

	for _, schedule := range fp.schedules {
		if schedule.IsDue(before) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}

	return dueSchedules, nil
}

// DeleteSchedule removes a schedule by its ID.
func (fp *FilePersistence) DeleteSchedule(id string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	delete(fp.schedules, id)

	return fp.saveSchedulesToFile()
}

// DeleteScheduleBySourceID removes a schedule by its source ID.
func (fp *FilePersistence) DeleteScheduleBySourceID(sourceID string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	for id, schedule := range fp.schedules {
		if schedule.SourceID == sourceID {
			delete(fp.schedules, id)

			break
		}
	}

	return fp.saveSchedulesToFile()
}

// HealthCheck verifies that the persistence layer is healthy.
func (fp *FilePersistence) HealthCheck() error {
	// Check if data directory is accessible
	if _, err := os.Stat(fp.dataDir); os.IsNotExist(err) {
		return fmt.Errorf("data directory does not exist: %s", fp.dataDir)
	}

	return nil
}

// Close cleans up resources.
func (fp *FilePersistence) Close() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	return fp.saveSchedulesToFile()
}

// loadSchedules loads schedules from the file system.
func (fp *FilePersistence) loadSchedules() error {
	schedulesFile := filepath.Join(fp.dataDir, "schedules.json")

	if _, err := os.Stat(schedulesFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty schedules
		return nil
	}

	data, err := os.ReadFile(schedulesFile) // #nosec G304 -- schedulesFile is constructed from controlled dataDir
	if err != nil {
		return fmt.Errorf("failed to read schedules file: %w", err)
	}

	var schedules []*models.Schedule
	if err := json.Unmarshal(data, &schedules); err != nil {
		return fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	// Convert to map
	for _, schedule := range schedules {
		fp.schedules[schedule.ID] = schedule
	}

	return nil
}

// saveSchedulesToFile saves all schedules to the file system.
func (fp *FilePersistence) saveSchedulesToFile() error {
	schedulesFile := filepath.Join(fp.dataDir, "schedules.json")

	// Convert map to slice
	schedules := make([]*models.Schedule, 0, len(fp.schedules))
	for _, schedule := range fp.schedules {
		schedules = append(schedules, schedule)
	}

	data, err := json.MarshalIndent(schedules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedules: %w", err)
	}

	if err := os.WriteFile(schedulesFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write schedules file: %w", err)
	}

	return nil
}
