package kafka

type EventTrigger struct {
	ID            string
	Topic         string
	ConsumerGroup string
	FilterRules   map[string]interface{}
}
