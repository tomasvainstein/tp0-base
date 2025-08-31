package common

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
	mu     sync.Mutex
	running bool
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		config: config,
		running: true,
		ctx:    ctx,
		cancel: cancel,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

func (c *Client) Stop() {
	log.Info("action: graceful_shutdown | result: in_progress")
	
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	
	c.cancel()
	
	c.cleanup()
	
	log.Info("action: graceful_shutdown | result: success")
}

func (c *Client) cleanup() {
	log.Info("action: cleanup | result: in_progress")
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn != nil {
		c.conn.Close()
		log.Info("action: cleanup | result: success | resource: client_connection")
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		c.mu.Lock()
		if !c.running {
			c.mu.Unlock()
			break
		}
		c.mu.Unlock()
		
		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		select {
		case <-time.After(c.config.LoopPeriod):
		case <-c.ctx.Done():
			log.Info("action: loop_interrupted | result: success | client_id: %v", c.config.ID)
			return
		}
	}
	
	c.mu.Lock()
	if c.running {
		log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
	} else {
		log.Info("action: loop_interrupted | result: success | client_id: %v", c.config.ID)
	}
	c.mu.Unlock()
}
