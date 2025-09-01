#!/bin/bash

# verificación de parámetros
if [ $# -ne 2 ]; then
    echo "se usan $0 <archivo_salida> <cantidad_clientes>"
    exit 1
fi

archivo_salida="$1"
cantidad_clientes="$2"

cat > "$archivo_salida" << 'YAML'
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
    volumes:
      - ./server/config.ini:/config.ini
    networks:
      - testing_net
YAML

declare -a nombres=("Juan" "María" "Carlos" "Ana" "Luis")
declare -a apellidos=("Gómez" "López" "Martínez" "Rodríguez" "García")
declare -a documentos=("12345678" "23456789" "34567890" "45678901" "56789012")
declare -a nacimientos=("1990-01-01" "1985-05-15" "1992-08-20" "1988-12-10" "1995-03-25")
declare -a numeros=("1001" "2002" "3003" "4004" "5005")

for i in $(seq 1 $cantidad_clientes); do
    idx=$((i-1))

    cat >> "$archivo_salida" << YAML

  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL=DEBUG
      - NOMBRE=${nombres[$idx]}
      - APELLIDO=${apellidos[$idx]}
      - DOCUMENTO=${documentos[$idx]}
      - NACIMIENTO=${nacimientos[$idx]}
      - NUMERO=${numeros[$idx]}
    volumes:
      - ./client/config.yaml:/config.yaml
    networks:
      - testing_net
    depends_on:
      - server
YAML
done

cat >> "$archivo_salida" << 'YAML'

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
YAML

echo "El archivo $archivo_salida se genero exitosamente con $cantidad_clientes clientes"