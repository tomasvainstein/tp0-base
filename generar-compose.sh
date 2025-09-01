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

for i in $(seq 1 $cantidad_clientes); do
    cat >> "$archivo_salida" << YAML

  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL=DEBUG
      - CLI_NOMBRE=Tomas
      - CLI_APELLIDO=Vainstein
      - CLI_DOCUMENTO=00000000
      - CLI_NACIMIENTO=2002-03-03
      - CLI_NUMERO=0000
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