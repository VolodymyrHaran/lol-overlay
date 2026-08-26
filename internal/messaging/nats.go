package messaging

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Client struct {
	conn      *nats.Conn
	jetStream jetstream.JetStream
}

const (
	flushTimeout = 5 * time.Second
	drainTimeout = 5 * time.Second
)

func New(url string) (*Client, error) {
	conn, err := nats.Connect(url,
		nats.Name("lol-group-helper"),
		nats.DrainTimeout(drainTimeout),
		nats.DisconnectErrHandler(
			func(_ *nats.Conn, err error) {
				slog.Warn(
					"NATS disconnected",
					"error", err,
				)
			},
		),
		nats.ReconnectHandler(
			func(conn *nats.Conn) {
				slog.Info(
					"NATS reconnected",
					"url", conn.ConnectedUrlRedacted(),
				)
			},
		),
		nats.ClosedHandler(
			func(conn *nats.Conn) {
				if err := conn.LastError(); err != nil {
					slog.Error(
						"NATS connection closed with error",
						"error", err,
					)
					return
				}
				slog.Info("NATS connection closed")
			},
		),
		nats.ErrorHandler(
			func(
				_ *nats.Conn,
				subscription *nats.Subscription,
				err error,
			) {
				subject := ""
				if subscription != nil {
					subject = subscription.Subject
				}
				slog.Error(
					"NATS asynchronous error",
					"subject", subject,
					"error", err,
				)
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()

		return nil, fmt.Errorf(
			"create JetStream client: %w",
			err,
		)
	}

	return &Client{
		conn:      conn,
		jetStream: js,
	}, nil
}

func (c *Client) Publish(subject string, data []byte) error {
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish %q: %w", subject, err)
	}

	return nil
}

func (c *Client) Subscribe(
	subject string,
	handler nats.MsgHandler,
) (*nats.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe %q: %w", subject, err)
	}

	return sub, nil
}

func (c *Client) Close() {
	if c == nil || c.conn == nil {
		return
	}

	c.conn.Close()
}

func (c *Client) Flush() error {
	if err := c.conn.FlushTimeout(
		flushTimeout,
	); err != nil {
		return fmt.Errorf(
			"flush NATS connection: %w",
			err,
		)
	}

	return nil
}

func (c *Client) Drain() error {
	if c == nil || c.conn == nil {
		return nil
	}

	if err := c.conn.Drain(); err != nil {
		return fmt.Errorf(
			"drain NATS connection: %w",
			err,
		)
	}

	return nil
}
