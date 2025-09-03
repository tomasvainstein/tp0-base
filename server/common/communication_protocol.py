import logging
from typing import Tuple, Optional
from .utils import Bet

MSG_TYPE_BET = 1
MSG_TYPE_ACK = 2
MSG_TYPE_FINISH_NOTIFICATION = 3
MSG_TYPE_WINNER_QUERY = 4
MSG_TYPE_WINNER_RESPONSE = 5

log = logging.getLogger(__name__)

def send_all(conn, data: bytes) -> bool:
    total_sent = 0
    while total_sent < len(data):
        try:
            sent = conn.send(data[total_sent:])
            if sent == 0:
                log.error("Connection broken during send")
                return False
            total_sent += sent
        except Exception as e:
            log.error(f"Error sending data: {e}")
            return False
    return True

def read_all(conn, length: int) -> Optional[bytes]:
    data = bytearray()
    while len(data) < length:
        try:
            chunk = conn.recv(length - len(data))
            if not chunk:
                log.error("Connection closed unexpectedly")
                return None
            data.extend(chunk)
        except Exception as e:
            log.error(f"Error reading data: {e}")
            return None
    return bytes(data)

def read_message(conn) -> Optional[Tuple[int, bytes]]:
    header = read_all(conn, 5)
    if header is None:
        return None
    
    msg_type = header[0]
    msg_length = (header[1] << 24) | (header[2] << 16) | (header[3] << 8) | header[4]
    
    payload = read_all(conn, msg_length)
    if payload is None:
        return None
    
    return msg_type, payload

def send_ack(conn, success: bool = True, error_msg: str = "") -> bool:
    if success:
        payload = b"OK"
    else:
        payload = f"ERROR: {error_msg}".encode('utf-8')
    
    msg_type = MSG_TYPE_ACK
    msg_length = len(payload)
    
    header = bytearray(5)
    header[0] = msg_type
    header[1] = (msg_length >> 24) & 0xFF
    header[2] = (msg_length >> 16) & 0xFF
    header[3] = (msg_length >> 8) & 0xFF
    header[4] = msg_length & 0xFF
    
    if not send_all(conn, bytes(header)):
        return False
    
    if not send_all(conn, payload):
        return False
    
    return True

def parse_bet_payload(payload: bytes) -> Optional[Bet]:
    try:
        payload_str = payload.decode('utf-8')
        
        fields = payload_str.split('|')
        if len(fields) != 5:
            log.error(f"Invalid bet payload format: expected 5 fields, got {len(fields)}")
            return None
        
        nombre, apellido, documento, nacimiento, numero = fields
        bet = Bet(
            agency="1",
            first_name=nombre,
            last_name=apellido,
            document=documento,
            birthdate=nacimiento,
            number=numero
        )
        
        return bet
        
    except Exception as e:
        log.error("Error parsing bet payload: {e}")
        return None

def parse_bet_batch_payload(payload: bytes) -> Optional[list]:
    try:
        payload_str = payload.decode('utf-8')
        lines = payload_str.strip().split('\n')
        
        if len(lines) < 2:
            log.error("Invalid batch payload: expected at least 2 lines")
            return None
        
        try:
            expected_count = int(lines[0])
        except ValueError:
            log.error("Invalid batch payload: first line must be a number")
            return None
        
        if len(lines) - 1 != expected_count:
            log.error(f"Invalid batch payload: expected {expected_count} bets, got {len(lines) - 1}")
            return None
        
        bets = []
        for i, line in enumerate(lines[1:], 1):
            fields = line.split('|')
            if len(fields) != 5:
                log.error(f"Invalid bet format at line {i}: expected 5 fields, got {len(fields)}")
                return None
            
            nombre, apellido, documento, nacimiento, numero = fields
            bet = Bet(
                agency="1",
                first_name=nombre,
                last_name=apellido,
                document=documento,
                birthdate=nacimiento,
                number=numero
            )
            bets.append(bet)
        
        return bets
        
    except Exception as e:
        log.error(f"Error parsing batch payload: {e}")
        return None
