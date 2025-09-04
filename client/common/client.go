package common

import (
	"net"
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
	Nombre        string
	Apellido      string
	Documento     string
	Nacimiento    string
	Numero        string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
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
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Cleanup() {
	log.Info("action: cleanup | result: in_progress")

	if c.conn != nil {
		c.conn.Close()
		log.Info("action: cleanup | result: success | resource: client_connection")
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	bet := NewBet(c.config.Nombre, c.config.Apellido, c.config.Documento, c.config.Nacimiento, c.config.Numero)

	if err := bet.Validate(); err != nil {
		log.Errorf("action: bet_validation | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	log.Infof("action: bet_created | result: success | client_id: %v | dni: %s | numero: %s",
		c.config.ID, bet.Documento, bet.Numero)

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an
		if err := c.createClientSocket(); err != nil {
			log.Errorf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
			continue
		}

		err := c.sendBet(bet)
		if err != nil {
			log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
		}

		err = c.getAck()
		if err != nil {
			log.Errorf("action: receive_ack | result: fail | client_id: %v | error: %v", c.config.ID, err)
		} else {
			log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", bet.Documento, bet.Numero)
		}

		c.conn.Close()
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
