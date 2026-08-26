package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
}

const flushTimeout = 5 * time.Second

func New(url string) (*Client, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	return &Client{
		conn: conn,
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
