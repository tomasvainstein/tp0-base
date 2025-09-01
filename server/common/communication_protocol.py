import struct
import logging
from typing import Tuple, Optional
from .utils import Bet

MSG_TYPE_BET = 1
MSG_TYPE_ACK = 2

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
    msg_length = struct.unpack('>I', header[1:])[0]
    
    payload = read_all(conn, msg_length)
    if payload is None:
        return None
    
    return msg_type, payload

def send_ack(conn) -> bool:
    payload = b"OK"
    msg_type = MSG_TYPE_ACK
    msg_length = len(payload)
    
    header = struct.pack('>BI', msg_type, msg_length)
    
    if not send_all(conn, header):
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
