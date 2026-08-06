package websocket

import (
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
)

type Client struct {
	conn *gorilla.Conn

	writeMu sync.Mutex
}

func NewClient(conn *gorilla.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func (c *Client) WriteMessage(
	messageType int,
	data []byte,
) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.SetWriteDeadline(
		time.Now().Add(writeWait),
	); err != nil {
		return err
	}

	return c.conn.WriteMessage(
		messageType,
		data,
	)
}

func (c *Client) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.conn.Close()
}
