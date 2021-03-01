package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"strings"
	"time"
)

// LogLevel represents the log level
// LogView recognizes three log levels: Info, Warning and Error
// Warning and Error events can be highlighted
type LogLevel uint

const (
	// LogLevelInfo is default log
	LogLevelInfo = LogLevel(iota)
	// LogLevelWarning is the level for warnings
	LogLevelWarning
	// LogLevelError is the level for errors
	LogLevelError
	// LogLevelAll is used for building histograms only as a placeholder for all log levels
	LogLevelAll
)

// LogEvent that can be added to LogView.
// Contains following fields:
//
// - EventID - a string identifier of the event, used in event handlers, may contain only ASCII characters
//
// - Source - a source that produced the event, may contain only ASCII characters
//
// - Timestamp - an instant when the event was created/ingested
//
// - Level - the severity level of an event. Can be used to highlight errors and warnings
//
// - Message - the event contents
type LogEvent struct {
	EventID   string
	Source    string
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

func NewLogEvent(eventID string, message string) *LogEvent {
	// expand tabs to 4 spaces (not exactly how it should be done, but will work for now)
	msg := strings.Replace(message, "\t", "    ", -1)

	return &LogEvent{
		EventID: eventID,
		Level:   LogLevelInfo,
		Message: msg,
	}
}

// printString is the most dump printing function. It just prints the string starting at x,y with
// a given style. No checks whatsoever are performed
func printString(screen tcell.Screen, x int, y int, text string, style tcell.Style) {
	for i, c := range text {
		screen.SetCell(x+i, y, style, c)
	}
}

func formatValue(value int) string {
	if value < 1000 {
		return fmt.Sprintf("%4d", value)
	} else if value < 1_000_000 {
		return fmt.Sprintf("%2.1fK", float64(value)/1_000.0)
	} else if value < 1_000_000_000 {
		return fmt.Sprintf("%2.1fM", float64(value)/1_000_000.0)
	} else {
		return fmt.Sprintf("%2.1fB", float64(value)/1_000_000_000.0)
	}
}
