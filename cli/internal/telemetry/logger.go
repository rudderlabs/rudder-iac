package telemetry

import (
	"fmt"

	"github.com/rudderlabs/analytics-go/v4"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

// CLI deals with only the internal logger interface
// so we need to write an adapter to make sure analytics client
// can log using our logger
var _ analytics.Logger = NewTelemetryLogger()

type TelemetryLogger struct {
	log *logger.Logger
}

func (l *TelemetryLogger) Logf(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *TelemetryLogger) Errorf(format string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(format, args...))
}

func NewTelemetryLogger() *TelemetryLogger {
	return &TelemetryLogger{
		log: logger.New("telemetry"),
	}
}
