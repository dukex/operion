package log

import (
	"context"

	logrus "github.com/sirupsen/logrus"
)

type contextKey string

const loggerKey contextKey = "logger"

func CreateContextWithLogger(logger *logrus.Entry) (context.Context, context.CancelFunc) {

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, loggerKey, logger)

	return ctx, cancel
}
