#!/bin/bash

# mensaje de prueba
TEST_MESSAGE="hello world"

echo "Probando echo server con mensaje de prueba '$TEST_MESSAGE'"

RESULT=$(docker run --rm --network tp0_testing_net busybox sh -c "
    # Enviar mensaje al servidor
    echo '$TEST_MESSAGE' | nc server 12345")

# verificacion de la respuesta del servidor
if [ "$RESULT" = "$TEST_MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
