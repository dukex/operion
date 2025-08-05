// Package gochannel provides in-memory channel implementation for testing and development.
package gochannel

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

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
