package schedule

import (
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/protocol"
)

type ScheduleReceiverFactory struct{}

func NewScheduleReceiverFactory() *ScheduleReceiverFactory {
	return &ScheduleReceiverFactory{}
}

func (f *ScheduleReceiverFactory) Create(config protocol.ReceiverConfig, eventBus eventbus.EventBus, logger *slog.Logger) (protocol.Receiver, error) {
	receiver := NewScheduleReceiver(eventBus, logger)
	if err := receiver.Configure(config); err != nil {
		return nil, err
	}
	return receiver, nil
}

func (f *ScheduleReceiverFactory) Type() string {
	return "schedule"
}

func (f *ScheduleReceiverFactory) Name() string {
	return "Schedule Receiver"
}

func (f *ScheduleReceiverFactory) Description() string {
	return "Receiver for handling cron-based schedule triggers and publishing trigger events"
}