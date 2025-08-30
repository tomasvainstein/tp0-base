import socket
import logging


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
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode('utf-8')
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')
            # TODO: Modify the send to avoid short-writes
            client_sock.send("{}\n".format(msg).encode('utf-8'))
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
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
