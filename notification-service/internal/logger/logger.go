package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

type EventLog struct {
	Time    string      `json:"time"`
	Subject string      `json:"subject"`
	Event   interface{} `json:"event"`
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) LogEvent(subject string, event interface{}) {
	logEntry := EventLog{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Subject: subject,
		Event:   event,
	}

	jsonOutput, err := json.Marshal(logEntry)
	if err != nil {
		fmt.Printf("{\"error\": \"failed to marshal log entry: %v\"}\n", err)
		return
	}

	fmt.Println(string(jsonOutput))
}
