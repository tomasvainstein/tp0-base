package common

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
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
	Nombre        string
	Apellido      string
	Documento     string
	Nacimiento    string
	Numero        string
	BatchMaxAmount int
	CSVFilePath   string
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
		return err
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

func (c *Client) processCSVBets(processBatch func([]*Bet) error) error {
	file, err := os.Open(c.config.CSVFilePath)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 5
	
	var currentBatch []*Bet
	lineNumber := 0
	totalBets := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Warningf("action: csv_validation | result: skip | line: %d | error: %v", lineNumber+1, err)
			continue
		}

		lineNumber++
		
		if len(record) == 0 || (len(record) == 1 && record[0] == "") {
			continue
		}

		bet := NewBet(record[0], record[1], record[2], record[3], record[4])
		if err := bet.Validate(); err != nil {
			log.Warningf("action: bet_validation | result: skip | line: %d | error: %v", lineNumber, err)
			continue
		}

		currentBatch = append(currentBatch, bet)
		totalBets++

		if len(currentBatch) >= c.config.BatchMaxAmount {
			if err := processBatch(currentBatch); err != nil {
				return fmt.Errorf("error processing batch: %v", err)
			}
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 {
		if err := processBatch(currentBatch); err != nil {
			return fmt.Errorf("error processing final batch: %v", err)
		}
	}

	log.Infof("action: csv_processed | result: success | total_bets: %d", totalBets)
	return nil
}

func (c *Client) StartClientLoop() {
	batchCount := 0
	
	processBatch := func(batch []*Bet) error {
		c.mu.Lock()
		if !c.running {
			c.mu.Unlock()
			return fmt.Errorf("client stopped")
		}
		c.mu.Unlock()
		
		if batchCount >= c.config.LoopAmount {
			return fmt.Errorf("reached maximum batch count")
		}
		
		if err := c.createClientSocket(); err != nil {
			log.Errorf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return err
		}

		err := c.sendBetBatch(batch)
		if err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | batch_size: %d | error: %v", c.config.ID, len(batch), err)
		} else {
			log.Infof("action: batch_sent | result: success | client_id: %v | batch_size: %d", c.config.ID, len(batch))
		}
		
		err = c.getAck()
		if err != nil {
			log.Errorf("action: receive_ack | result: fail | client_id: %v | batch_size: %d | error: %v", c.config.ID, len(batch), err)
		} else {
			log.Infof("action: apuesta_enviada | result: success | client_id: %v | batch_size: %d", c.config.ID, len(batch))
		}
		
		c.conn.Close()
		batchCount++

		select {
		case <-time.After(c.config.LoopPeriod):
		case <-c.ctx.Done():
			log.Infof("action: loop_interrupted | result: success | client_id: %v", c.config.ID)
			return fmt.Errorf("context cancelled")
		}
		
		return nil
	}
	
	err := c.processCSVBets(processBatch)
	if err != nil {
		log.Errorf("action: csv_processing | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	
	c.mu.Lock()
	if c.running {
		log.Infof("action: loop_finished | result: success | client_id: %v | batches_sent: %d", c.config.ID, batchCount)
	} else {
		log.Infof("action: loop_interrupted | result: success | client_id: %v | batches_sent: %d", c.config.ID, batchCount)
	}
	c.mu.Unlock()
}
