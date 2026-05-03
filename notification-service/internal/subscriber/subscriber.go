package subscriber

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type NotificationEvent struct {
	Time    string      `json:"time"`
	Subject string      `json:"subject"`
	Event   interface{} `json:"event"`
}

type Subscriber struct {
	nc *nats.Conn
}

func NewSubscriber(nc *nats.Conn) *Subscriber {
	return &Subscriber{nc: nc}
}

func (s *Subscriber) SubscribeToAll(subjects []string) error {
	for _, subject := range subjects {
		_, err := s.nc.Subscribe(subject, func(m *nats.Msg) {
			var eventData interface{}
			if err := json.Unmarshal(m.Data, &eventData); err != nil {
				log.Printf("Error unmarshaling event from %s: %v", m.Subject, err)
				return
			}

			output := NotificationEvent{
				Time:    time.Now().UTC().Format(time.RFC3339),
				Subject: m.Subject,
				Event:   eventData,
			}

			jsonOutput, _ := json.Marshal(output)
			fmt.Println(string(jsonOutput))
		})
		if err != nil {
			return fmt.Errorf("error subscribing to %s: %w", subject, err)
		}
	}
	return nil
}
