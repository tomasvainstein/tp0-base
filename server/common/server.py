import socket
import logging
from .communication_protocol import read_message, send_ack, parse_bet_payload, parse_bet_batch_payload, MSG_TYPE_BET, MSG_TYPE_FINISH_NOTIFICATION, MSG_TYPE_WINNER_QUERY, MSG_TYPE_WINNER_RESPONSE, send_winner_response
from .utils import store_bets, load_bets, has_won, LOTTERY_WINNER_NUMBER

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        self._finished_agencies = set()
        self._sorteo_realizado = False
        self._pending_queries = []

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
            
            if msg_type == MSG_TYPE_BET:
                self.__handle_bet_message(client_sock, payload)
            elif msg_type == MSG_TYPE_FINISH_NOTIFICATION:
                self.__handle_finish_notification(client_sock, payload)
            elif msg_type == MSG_TYPE_WINNER_QUERY:
                self.__handle_winner_query(client_sock, payload)
            else:
                logging.error(f"action: receive_message | result: fail | error: unexpected message type {msg_type}")
                return
            
        except OSError as e:
            logging.error(f"action: handle_connection | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __handle_bet_message(self, client_sock, payload):
        bets = parse_bet_batch_payload(payload)
        if bets is None:
            bet = parse_bet_payload(payload)
            if bet is None:
                logging.error("action: receive_message | result: fail | error: could not parse bet or batch")
                send_ack(client_sock, success=False, error_msg="Invalid payload format")
                return
            bets = [bet]
        
        bet_count = len(bets)
        logging.info(f'action: receive_message | result: success | ip: {client_sock.getpeername()[0]} | cantidad: {bet_count}')
        
        try:
            store_bets(bets)
            logging.info(f'action: apuesta_recibida | result: success | cantidad: {bet_count}')
            
            if not send_ack(client_sock, success=True):
                logging.error("action: send_ack | result: fail | error: could not send ACK")
                return
            
            logging.info("action: send_ack | result: success")
            
        except Exception as e:
            logging.error(f'action: apuesta_recibida | result: fail | cantidad: {bet_count} | error: {e}')
            send_ack(client_sock, success=False, error_msg=f"Failed to store bets: {e}")
            return

    def __handle_finish_notification(self, client_sock, payload):
        try:
            agency_id = payload.decode('utf-8')
            logging.info(f'action: finish_notification | result: success | agency: {agency_id}')
            
            self._finished_agencies.add(agency_id)

            if len(self._finished_agencies) == 5 and not self._sorteo_realizado:
                logging.info('action: sorteo | result: success')
                self._sorteo_realizado = True

                self.__process_pending_queries()
            
            if not send_ack(client_sock, success=True):
                logging.error("action: send_ack | result: fail | error: could not send ACK")
                return
            
            logging.info("action: send_ack | result: success")
            
        except Exception as e:
            logging.error(f'action: finish_notification | result: fail | error: {e}')
            send_ack(client_sock, success=False, error_msg=f"Failed to process notification: {e}")

    def __process_pending_queries(self):

        logging.info(f'action: process_pending_queries | result: in_progress | pending_count: {len(self._pending_queries)}')
        
        for client_sock, agency_id in self._pending_queries:
            try:
                self.__process_winner_query(client_sock, agency_id)
            except Exception as e:
                logging.error(f'action: process_pending_query | result: fail | agency: {agency_id} | error: {e}')
                send_winner_response(client_sock, 0)
        
        self._pending_queries.clear()
        logging.info('action: process_pending_queries | result: success')

    def __handle_winner_query(self, client_sock, payload):

        try:
            agency_id = payload.decode('utf-8')
            
            if not self._sorteo_realizado:
                logging.info(f'action: winner_query | result: pending | agency: {agency_id} | reason: sorteo not performed yet')
                self._pending_queries.append((client_sock, agency_id))
                return
            
            self.__process_winner_query(client_sock, agency_id)
            
        except Exception as e:
            logging.error(f'action: winner_query | result: fail | error: {e}')
            send_winner_response(client_sock, 0)

    def __process_winner_query(self, client_sock, agency_id):
        try:
            try:
                all_bets = list(load_bets())
                agency_bets = [bet for bet in all_bets if str(bet.agency) == agency_id]
            except FileNotFoundError:
                agency_bets = []

            winners = [bet for bet in agency_bets if has_won(bet)]
            winner_count = len(winners)

            logging.info(f'action: winner_query | result: success | agency: {agency_id} | winners: {winner_count}')
            
            if not send_winner_response(client_sock, winner_count):
                logging.error("action: send_winner_response | result: fail | error: could not send response")
                return
            
            logging.info("action: send_winner_response | result: success")
            
        except Exception as e:
            logging.error(f'action: winner_query | result: fail | error: {e}')
            send_winner_response(client_sock, 0)

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
