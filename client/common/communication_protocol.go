package common

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	MSG_TYPE_BET = 1
	MSG_TYPE_ACK = 2
)

type Message struct {
	Type    uint8
	Length  uint32
	Payload []byte
}

func NewMessage(msgType uint8, payload []byte) *Message {
	return &Message{
		Type:    msgType,
		Length:  uint32(len(payload)),
		Payload: payload,
	}
}

func sendAll(conn net.Conn, data []byte) error {
	totalSent := 0
	for totalSent < len(data) {
		n, err := conn.Write(data[totalSent:])
		if err != nil {
			return fmt.Errorf("error writing to connection: %v", err)
		}
		totalSent += n
	}
	return nil
}

func readAll(conn net.Conn, length uint32) ([]byte, error) {
	data := make([]byte, length)
	totalRead := uint32(0)
	
	for totalRead < length {
		n, err := conn.Read(data[totalRead:])
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("connection closed unexpectedly, expected %d bytes, got %d", length, totalRead)
			}
			return nil, fmt.Errorf("error reading from connection: %v", err)
		}
		totalRead += uint32(n)
	}
	
	return data, nil
}

func (c *Client) sendBet(bet *Bet) error {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s",
		bet.Nombre, bet.Apellido, bet.Documento, bet.Nacimiento, bet.Numero)
	
	msg := NewMessage(MSG_TYPE_BET, []byte(payload))
	
	header := make([]byte, 5)
	header[0] = msg.Type
	binary.BigEndian.PutUint32(header[1:], msg.Length)
	
	if err := sendAll(c.conn, header); err != nil {
		return fmt.Errorf("error sending header: %v", err)
	}
	
	if err := sendAll(c.conn, msg.Payload); err != nil {
		return fmt.Errorf("error sending payload: %v", err)
	}
	
	return nil
}

func (c *Client) sendBetBatch(bets []*Bet) error {
	if len(bets) == 0 {
		return fmt.Errorf("cannot send empty batch")
	}
	
	payload := fmt.Sprintf("%d\n", len(bets))
	for _, bet := range bets {
		betStr := fmt.Sprintf("%s|%s|%s|%s|%s\n",
			bet.Nombre, bet.Apellido, bet.Documento, bet.Nacimiento, bet.Numero)
		payload += betStr
	}
	
	msg := NewMessage(MSG_TYPE_BET, []byte(payload))
	
	header := make([]byte, 5)
	header[0] = msg.Type
	binary.BigEndian.PutUint32(header[1:], msg.Length)
	
	if err := sendAll(c.conn, header); err != nil {
		return fmt.Errorf("error sending header: %v", err)
	}
	
	if err := sendAll(c.conn, msg.Payload); err != nil {
		return fmt.Errorf("error sending payload: %v", err)
	}
	
	return nil
}

func (c *Client) getAck() error {
	header, err := readAll(c.conn, 5)
	if err != nil {
		return fmt.Errorf("error reading header: %v", err)
	}
	
	msgType := header[0]
	msgLength := binary.BigEndian.Uint32(header[1:])
	
	if msgType != MSG_TYPE_ACK {
		return fmt.Errorf("unexpected message type: expected %d, got %d", MSG_TYPE_ACK, msgType)
	}
	
	payload, err := readAll(c.conn, msgLength)
	if err != nil {
		return fmt.Errorf("error reading ACK payload: %v", err)
	}
	
	response := string(payload)
	if response != "OK" {
		return fmt.Errorf("server error: %s", response)
	}
	
	return nil
}
