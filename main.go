package timeteller

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Interrupt)
	defer cancel()

	topicID := "timeline-v1"

	gcpCredentials, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		log.Fatalf("google::FindDefaultCredentials: %v", err)
	}

	projectID := gcpCredentials.ProjectID //os.Getenv("GCP_PROJECT_ID")

	err = createTopic(ctx, projectID, topicID)
	if err != nil {
		log.Fatalf("createTopic: %v", err)
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ok!"))
	})

	tickerTape := newTickerTape(projectID, topicID)
	router.Handle("/api/ticker/start", otelhttp.NewHandler(tickerTape, "api/ticker/start")).Methods("POST")

	port := os.Getenv("PORT")
	srv := http.Server{
		Handler:      router,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Received shutdown signal, shutting down..")
		srv.Shutdown(context.Background())
	}()

	fmt.Println("Listening on " + port)
	log.Fatal(srv.ListenAndServe())
}

func createTopic(ctx context.Context, projectID, topicID string) error {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("%w: failed to setup pubsub client", err)
	}

	topic := client.TopicInProject(topicID, projectID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to check if topic exists", err)
	}
	if exists {
		fmt.Printf("Topic %s already exists\n", topicID)
		return nil
	}

	_, err = client.CreateTopic(ctx, topicID)
	if err != nil {
		return fmt.Errorf("%w: failed to create topic", err)
	}

	fmt.Printf("Topic created: %s\n", topicID)
	return nil
}
