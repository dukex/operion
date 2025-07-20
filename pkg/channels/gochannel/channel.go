// Package gochannel provides in-memory channel implementation for testing and development.
package gochannel

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// CreateChannel creates a GoChannel-based publisher and subscriber for testing
// This is ideal for unit tests and local development as it doesn't require external dependencies
func CreateChannel(logger watermill.LoggerAdapter) (*gochannel.GoChannel, *gochannel.GoChannel, error) {
	// GoChannel pubsub is the same instance for both publisher and subscriber
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            1000,  // Buffer size for output channels
			Persistent:                     false, // Don't persist messages after consumption
			BlockPublishUntilSubscriberAck: false, // Don't block on publish
		},
		logger,
	)

	// Return the same instance for both publisher and subscriber
	// GoChannel implements both Publisher and Subscriber interfaces
	return pubSub, pubSub, nil
}

// CreateTestChannel creates a minimal GoChannel setup for testing
// with smaller buffers and blocking behavior for deterministic tests
func CreateTestChannel(logger watermill.LoggerAdapter) (*gochannel.GoChannel, *gochannel.GoChannel, error) {
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            10,   // Smaller buffer for tests
			Persistent:                     true, // Keep messages for inspection in tests
			BlockPublishUntilSubscriberAck: true, // Block until acknowledged for deterministic behavior
		},
		logger,
	)

	return pubSub, pubSub, nil
}
