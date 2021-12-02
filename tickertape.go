package timeteller

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type tickerTape struct {
	projectID string
	topicID   string
}

func newTickerTape(projectID string, topicID string) tickerTape {
	return tickerTape{
		projectID: projectID,
		topicID:   topicID,
	}
}

func (t tickerTape) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	qDuration := r.URL.Query().Get("duration")

	duration, err := strconv.Atoi(qDuration)
	if err != nil {
		duration = 30
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case time := <-ticker.C:
				_, err := t.publishTime(time, r.Context())
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
	}()

	time.Sleep(time.Duration(duration) * time.Millisecond)
	ticker.Stop()
	done <- true
	fmt.Println("Ticker stopped")

	w.WriteHeader(http.StatusOK)
	return
}

func (t tickerTape) publishTime(time time.Time, ctx context.Context) (string, error) {
	client, err := pubsub.NewClient(ctx, t.projectID)
	if err != nil {
		return "", fmt.Errorf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	topic := client.Topic(t.topicID)
	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte(fmt.Sprintf("Current time %v", time)),
	})
	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	id, err := result.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("Get: %v", err)
	}
	return id, nil
}
