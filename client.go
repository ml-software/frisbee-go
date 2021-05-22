package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"net"
)

// ClientRouterFunc defines a message handler type
type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)

// ClientRouter defines map of message handlers
type ClientRouter map[uint32]ClientRouterFunc

// Client accepts and handles inbound messages
type Client struct {
	addr    string
	Conn    *Conn
	router  ClientRouter
	Options *Options
	closed  bool
}

// NewClient returns an initialized client
func NewClient(addr string, router ClientRouter, opts ...Option) *Client {

	options := loadOptions(opts...)

	newRouter := router

	if options.Heartbeat {
		newRouter[0] = HandleHeartbeat
		for message, handler := range router {
			newRouter[message+1] = handler
		}
	}

	return &Client{
		addr:    addr,
		router:  newRouter,
		Options: options,
		closed:  false,
	}
}

func (c *Client) Connect() error {
	c.logger().Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := Connect("tcp", c.addr, c.Options.KeepAlive, c.logger())
	if err != nil {
		return err
	}
	c.Conn = frisbeeConn
	c.logger().Info().Msgf("Connected to %s", c.addr)

	go c.reactor()

	c.logger().Debug().Msgf("Reactor started for %s", c.addr)

	return nil
}

func (c *Client) Close() error {
	c.closed = true
	return c.Conn.Close()
}

func (c *Client) Write(message *Message, content *[]byte) error {
	return c.Conn.Write(message, content)
}

func (c *Client) Raw() (net.Conn, error) {
	if c.Conn == nil {
		return nil, ConnectionNotInitialized
	}
	c.closed = true
	return c.Conn.Raw(), nil
}

func (c *Client) logger() *zerolog.Logger {
	return c.Options.Logger
}

func (c *Client) reactor() {
	for {
		incomingMessage, incomingContent, err := c.Conn.Read()
		if err != nil {
			c.logger().Error().Msgf(errors.WithContext(err, CLOSE).Error())
			_ = c.Close()
			return
		}

		routerFunc := c.router[incomingMessage.Operation]
		if routerFunc != nil {
			var outgoingMessage *Message
			var outgoingContent []byte
			var action Action
			if incomingMessage.ContentLength == 0 || incomingContent == nil {
				outgoingMessage, outgoingContent, action = routerFunc(*incomingMessage, nil)
			} else {
				outgoingMessage, outgoingContent, action = routerFunc(*incomingMessage, *incomingContent)
			}

			if outgoingMessage != nil && outgoingMessage.ContentLength == uint64(len(outgoingContent)) {
				err := c.Conn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					c.logger().Error().Msgf(errors.WithContext(err, CLOSE).Error())
					_ = c.Close()
					return
				}
			}

			switch action {
			case Close:
				c.logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.addr)
				_ = c.Close()
				return
			case Shutdown:
				c.logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.addr)
				_ = c.Close()
				return
			default:
			}
		}
	}
}
