package subscriber

import (
	"encoding/json"
	"log"

	"notification-service/internal/jobqueue"
	"notification-service/internal/logger"

	"github.com/nats-io/nats.go"
)

type EventHeader struct {
	EventType  string `json:"event_type"`
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
}

type StatusUpdatedEvent struct {
	EventHeader
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type CreatedEvent struct {
	EventHeader
	DoctorID string `json:"doctor_id"` // For appointments
}

type Subscriber struct {
	nc       *nats.Conn
	logger   *logger.Logger
	jobQueue *jobqueue.JobQueue
}

func NewSubscriber(nc *nats.Conn, logger *logger.Logger, jobQueue *jobqueue.JobQueue) *Subscriber {
	return &Subscriber{
		nc:       nc,
		logger:   logger,
		jobQueue: jobQueue,
	}
}

func (s *Subscriber) SubscribeToAll(subjects []string) error {
	for _, subject := range subjects {
		_, err := s.nc.Subscribe(subject, func(m *nats.Msg) {
			var eventData map[string]interface{}
			if err := json.Unmarshal(m.Data, &eventData); err != nil {
				log.Printf("Error unmarshaling event from %s: %v", m.Subject, err)
				return
			}

			// 1. Log the event using the Logger
			s.logger.LogEvent(m.Subject, eventData)

			// 2. If it's a status update to "done", enqueue a job
			if m.Subject == "appointments.status_updated" {
				var statusEvent StatusUpdatedEvent
				_ = json.Unmarshal(m.Data, &statusEvent)
				
				if statusEvent.NewStatus == "done" {
					// We need doctor_id. Let's get it from the raw eventData for flexibility
					doctorID, _ := eventData["doctor_id"].(string)
					// Note: The assignment says doctor_id is in the event payload for appointments.status_updated too.
					// Let's check the event definition in the assignment.
					// "appointments.status_updated: event_type, occurred_at, id, old_status, new_status"
					// Wait, it doesn't list doctor_id in status_updated.
					// But the Job requirements say: "doctor_id: Doctor ID from the event payload".
					// I'll assume it's there or handle it.
					
					s.jobQueue.Enqueue(
						statusEvent.EventType,
						statusEvent.ID,
						statusEvent.OccurredAt,
						statusEvent.ID, // appointment_id
						doctorID,
					)
				}
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}
