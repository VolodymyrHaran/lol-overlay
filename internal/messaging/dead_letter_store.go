package messaging

import (
	"context"
	"encoding/json"
	"fmt"
)

type GameDeadLetterRecord struct {
	Sequence uint64
	Event    GameDeadLetterEvent
}

func (c *Client) GetGameDeadLetter(
	ctx context.Context,
	sequence uint64,
) (GameDeadLetterEvent, error) {
	if sequence == 0 {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"dead-letter sequence must be positive",
		)
	}

	stream, err := c.jetStream.Stream(
		ctx,
		StreamGameEventsDLQ,
	)
	if err != nil {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"get dead-letter stream: %w",
			err,
		)
	}

	message, err := stream.GetMsg(
		ctx,
		sequence,
	)
	if err != nil {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"get dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	return decodeGameDeadLetter(
		message.Sequence,
		message.Subject,
		message.Data,
	)
}

func (c *Client) DeleteGameDeadLetter(
	ctx context.Context,
	sequence uint64,
) error {
	if sequence == 0 {
		return fmt.Errorf(
			"dead-letter sequence must be positive",
		)
	}

	stream, err := c.jetStream.Stream(
		ctx,
		StreamGameEventsDLQ,
	)
	if err != nil {
		return fmt.Errorf(
			"get dead-letter stream: %w",
			err,
		)
	}

	if err := stream.DeleteMsg(
		ctx,
		sequence,
	); err != nil {
		return fmt.Errorf(
			"delete dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	return nil
}

func (c *Client) GetLastGameDeadLetter(
	ctx context.Context,
) (GameDeadLetterRecord, error) {
	stream, err := c.jetStream.Stream(
		ctx,
		StreamGameEventsDLQ,
	)
	if err != nil {
		return GameDeadLetterRecord{}, fmt.Errorf(
			"get dead-letter stream: %w",
			err,
		)
	}

	message, err := stream.GetLastMsgForSubject(
		ctx,
		SubjectGameDeadLetter,
	)
	if err != nil {
		return GameDeadLetterRecord{}, fmt.Errorf(
			"get latest dead-letter message: %w",
			err,
		)
	}

	event, err := decodeGameDeadLetter(
		message.Sequence,
		message.Subject,
		message.Data,
	)
	if err != nil {
		return GameDeadLetterRecord{}, err
	}

	return GameDeadLetterRecord{
		Sequence: message.Sequence,
		Event:    event,
	}, nil
}

func decodeGameDeadLetter(
	sequence uint64,
	subject string,
	data []byte,
) (GameDeadLetterEvent, error) {
	if subject != SubjectGameDeadLetter {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"unexpected dead-letter subject %q",
			subject,
		)
	}

	var event GameDeadLetterEvent

	if err := json.Unmarshal(
		data,
		&event,
	); err != nil {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"decode dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	if err := validateGameDeadLetterEvent(
		event,
	); err != nil {
		return GameDeadLetterEvent{}, fmt.Errorf(
			"validate dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	return event, nil
}
