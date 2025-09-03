package common

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
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

func (c *Client) readCSVBets() ([]*Bet, error) {
	file, err := os.Open(c.config.CSVFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var bets []*Bet
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) != 5 {
			log.Warningf("action: csv_validation | result: skip | line: %d | error: invalid number of fields", lineNumber)
			continue
		}

		bet := NewBet(fields[0], fields[1], fields[2], fields[3], fields[4])
		if err := bet.Validate(); err != nil {
			log.Warningf("action: bet_validation | result: skip | line: %d | error: %v", lineNumber, err)
			continue
		}

		bets = append(bets, bet)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV file: %v", err)
	}

	log.Infof("action: csv_loaded | result: success | total_bets: %d", len(bets))
	return bets, nil
}

func (c *Client) createBatches(bets []*Bet) [][]*Bet {
	var batches [][]*Bet
	
	for i := 0; i < len(bets); i += c.config.BatchMaxAmount {
		end := i + c.config.BatchMaxAmount
		if end > len(bets) {
			end = len(bets)
		}
		batches = append(batches, bets[i:end])
	}
	
	log.Infof("action: batches_created | result: success | total_batches: %d | max_batch_size: %d", 
		len(batches), c.config.BatchMaxAmount)
	return batches
}

func (c *Client) StartClientLoop() {
	bets, err := c.readCSVBets()
	if err != nil {
		log.Errorf("action: csv_loading | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	batches := c.createBatches(bets)
	
	batchCount := 0
	for batchCount < c.config.LoopAmount && batchCount < len(batches) {
		c.mu.Lock()
		if !c.running {
			c.mu.Unlock()
			break
		}
		c.mu.Unlock()
		
		batch := batches[batchCount]
		
		if err := c.createClientSocket(); err != nil {
			log.Errorf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
			batchCount++
			continue
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
			return
		}
	}
	
	c.mu.Lock()
	if c.running {
		log.Infof("action: loop_finished | result: success | client_id: %v | batches_sent: %d", c.config.ID, batchCount)
		
		if err := c.createClientSocket(); err != nil {
			log.Errorf("action: connect_for_notification | result: fail | client_id: %v | error: %v", c.config.ID, err)
		} else {
			if err := c.sendFinishNotification(); err != nil {
				log.Errorf("action: send_finish_notification | result: fail | client_id: %v | error: %v", c.config.ID, err)
			} else {
				log.Infof("action: finish_notification_sent | result: success | client_id: %v", c.config.ID)
			}
			
			if err := c.getAck(); err != nil {
				log.Errorf("action: receive_notification_ack | result: fail | client_id: %v | error: %v", c.config.ID, err)
			} else {
				log.Infof("action: notification_ack_received | result: success | client_id: %v", c.config.ID)
			}
			
			c.conn.Close()
			
			if err := c.createClientSocket(); err != nil {
				log.Errorf("action: connect_for_winner_query | result: fail | client_id: %v | error: %v", c.config.ID, err)
			} else {
				if err := c.sendWinnerQuery(); err != nil {
					log.Errorf("action: send_winner_query | result: fail | client_id: %v | error: %v", c.config.ID, err)
				} else {
					log.Infof("action: winner_query_sent | result: success | client_id: %v", c.config.ID)
					
					winnerCount, err := c.getWinnerResponse()
					if err != nil {
						log.Errorf("action: receive_winner_response | result: fail | client_id: %v | error: %v", c.config.ID, err)
					} else {
						log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", winnerCount)
						
						if winnerCount == 0 {
							log.Infof("action: waiting_for_sorteo | result: in_progress | client_id: %v", c.config.ID)
							time.Sleep(2 * time.Second)
							c.conn.Close()
							
							if err := c.createClientSocket(); err != nil {
								log.Errorf("action: connect_for_retry_winner_query | result: fail | client_id: %v | error: %v", c.config.ID, err)
							} else {
								if err := c.sendWinnerQuery(); err != nil {
									log.Errorf("action: retry_winner_query | result: fail | client_id: %v | error: %v", c.config.ID, err)
								} else {
									log.Infof("action: retry_winner_query_sent | result: success | client_id: %v", c.config.ID)
									
									winnerCount, err := c.getWinnerResponse()
									if err != nil {
										log.Errorf("action: receive_retry_winner_response | result: fail | client_id: %v | error: %v", c.config.ID, err)
									} else {
										log.Infof("action: consulta_ganadores_retry | result: success | cant_ganadores: %d", winnerCount)
									}
								}
								
								c.conn.Close()
							}
						}
					}
				}
				
				if c.conn != nil {
					c.conn.Close()
				}
			}
		}
	} else {
		log.Infof("action: loop_interrupted | result: success | client_id: %v | batches_sent: %d", c.config.ID, batchCount)
	}
	c.mu.Unlock()
}
