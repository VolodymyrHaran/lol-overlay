package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"lol-timer/internal/messaging"
	"lol-timer/internal/services"

	"github.com/joho/godotenv"
)

const commandTimeout = 15 * time.Second

func main() {
	var (
		sequence uint64
		latest   bool
		execute  bool
	)

	flag.Uint64Var(
		&sequence,
		"sequence",
		0,
		"JetStream sequence of the GAME_EVENTS_DLQ message",
	)

	flag.BoolVar(
		&latest,
		"latest",
		false,
		"show the latest GAME_EVENTS_DLQ message",
	)

	flag.BoolVar(
		&execute,
		"execute",
		false,
		"republish the original event and delete the DLQ message",
	)

	flag.Parse()

	if sequence == 0 && !latest {
		log.Fatal(
			"specify either -sequence or -latest",
		)
	}

	if sequence != 0 && latest {
		log.Fatal(
			"-sequence and -latest cannot be used together",
		)
	}

	if latest && execute {
		log.Fatal(
			"-execute requires an explicit -sequence",
		)
	}

	if err := godotenv.Load(); err != nil {
		log.Println(
			".env file not found, using system environment variables",
		)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		commandTimeout,
	)
	defer cancel()

	client, err := messaging.New(natsURL)
	if err != nil {
		log.Fatal("connect NATS: ", err)
	}
	defer client.Close()

	var event messaging.GameDeadLetterEvent

	if latest {
		record, err := client.GetLastGameDeadLetter(
			ctx,
		)
		if err != nil {
			log.Fatal(
				"load latest dead-letter message: ",
				err,
			)
		}

		sequence = record.Sequence
		event = record.Event
	} else {
		event, err = client.GetGameDeadLetter(
			ctx,
			sequence,
		)
		if err != nil {
			log.Fatal(
				"load dead-letter message: ",
				err,
			)
		}
	}

	log.Printf(
		"dead-letter message: sequence=%d event_id=%q "+
			"original_subject=%q source_stream=%q "+
			"source_sequence=%d consumer=%q "+
			"delivery_count=%d error=%q",
		sequence,
		event.EventID,
		event.OriginalSubject,
		event.SourceStream,
		event.StreamSequence,
		event.Consumer,
		event.DeliveryCount,
		event.Error,
	)

	if !execute {
		log.Println(
			"dry run only; pass the displayed " +
				"-sequence together with -execute to replay",
		)
		return
	}

	replayService :=
		services.NewDeadLetterReplayService(
			client,
			client,
		)

	result, err := replayService.Replay(
		ctx,
		sequence,
	)
	if err != nil {
		log.Fatal(
			"replay dead-letter message: ",
			err,
		)
	}

	log.Printf(
		"dead-letter replay completed: "+
			"sequence=%d event_id=%q subject=%q duplicate=%t",
		result.Sequence,
		result.EventID,
		result.Subject,
		result.Duplicate,
	)
}
