import socket
import logging
from .communication_protocol import read_message, send_ack, parse_bet_payload, MSG_TYPE_BET
from .utils import store_bets

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True

    def run(self):
        """
        Server loop with graceful shutdowns:

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        logging.info('action: accept_connections | result: in_progress')
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                if client_sock:
                    self.__handle_client_connection(client_sock)

            except Exception as e:
                if self._running:
                    logging.error("action: accept_connections | result: fail | error: {e}")
                break
        
        self._cleanup()

    def stop(self):
        """Detiene el servidor de forma graceful"""
        logging.info("action: stop | result: in_progress")
        self._running = False

    def _cleanup(self):
        """Limpia todos los recursos del servidor"""
        logging.info("action: cleanup | result: in_progress")
        
        try:
            if self._server_socket:
                self._server_socket.close()
                logging.info("action: cleanup | result: success | resource: server_socket")
        except Exception as e:
            logging.error("action: cleanup | result: fail | resource: server_socket | error: {e}")
        
        logging.info("action: graceful_shutdown | result: success")

    def __handle_client_connection(self, client_sock):
        """
        Maneja la conexión con el cliente usando el protocolo de lotería
        """
        try:
            # leer mensaje del cliente
            message = read_message(client_sock)
            if message is None:
                logging.error("action: receive_message | result: fail | error: could not read message")
                return
            
            msg_type, payload = message
            
            # verificar que sea un mensaje de apuesta
            if msg_type != MSG_TYPE_BET:
                logging.error(f"action: receive_message | result: fail | error: unexpected message type {msg_type}")
                return
            
            # parsear la apuesta
            bet = parse_bet_payload(payload)
            if bet is None:
                logging.error("action: receive_message | result: fail | error: could not parse bet")
                return
            
            logging.info(f'action: receive_message | result: success | ip: {client_sock.getpeername()[0]} | dni: {bet.document} | numero: {bet.number}')
            
            # guarda la apuesta
            store_bets([bet])
            
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
            
            # enviar ACK al cliente
            if not send_ack(client_sock):
                logging.error("action: send_ack | result: fail | error: could not send ACK")
                return
            
            logging.info("action: send_ack | result: success")
            
        except OSError as e:
            logging.error("action: handle_connection | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
