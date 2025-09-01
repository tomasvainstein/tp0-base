# TP0: Documentación

## Ejercicio 1
En la resolución del ejercicio se creó un script de bash con nombre `generar-compose.sh` que permite generar un archivo Docker Compose con una cantidad configurable de clientes, siguiendo el formato `client1`, `client2`, `client3`, etc.

Parae ejecutar el script de bash se usa el comando:

`./generar-compose.sh docker-compose-dev.yaml 5`, pero primero hay que darle permisos de ejecución `chmod +x generar-compose.sh`.

El script verifica que se proporcionen los 2 parámetros: archivo de salida y cantidad de clientes. Luego se mantiene la configuración del server y client que ya se encuentran en el archivo docker-compose-dev original (`container_name: server`, `image: server:latest` y `entrypoint: python3 /main.py`) y se hace un bucle para generar la cantidad de clientes especificada por parámetro (también se mantiene la estructura del archivo original para los clientes). Para la configuración de red se usan los valores `testing_net` y `172.25.125.0/24`.

Para levantar los contenedores, se usa el comando `make docker-compose-up` finalmente y se pueden verificar que los clientes se comunican correctamente con el server verificando los logs con `make docker-compose-logs`. 

## Ejercicio 2
En este ejercicio se actualizó el script de bash `generar-compose.sh` creado en el ejercicio 1 para incluir automáticamente los volúmenes en cada servicio generado.

De esta forma se modificó el cliente y el servidor para lograr que realizar cambios en los archivos de configuración no requiera reconstruir las imágenes de Docker. La configuración se inyecta en los contenedores a través de volumenes de Docker, permitiendo que los archivos persistan por fuera de la imagen y no sea necesario reiniciar o frenar la ejecución los contenedores para aplicar cambios de configuración.

## Ejercicio 5
En este ejercicio se implementó el sistema de apuestas de quiniela con comunicación cliente-servidor usando sockets del protocolo TCP.

El client se implementó en Go y emula una agencia de quiniela. Cada cliente recibe como variables de entorno los campos de la apuesta:
- `CLI_NOMBRE`: Nombre de la persona
- `CLI_APELLIDO`: Apellido de la persona  
- `CLI_DOCUMENTO`: DNI de la persona
- `CLI_NACIMIENTO`: Fecha de nacimiento
- `CLI_NUMERO`: Número apostado

Se crean apuestas con estos datos, se validan y se envían al servidor. Al recibir la confirmación del servidor se registra el log: `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

El server se implementó en Python y emula la Lotería Nacional. Recibe los campos de cada apuesta desde los clientes y se almacena la información mediante la función `store_bets` que ya fue provista por la cátedra. Al persistir cada apuesta se loggea: `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.


### Protocolo
El protocolo de comunicación utiliza un formato de mensaje con:

1. Header del Mensaje (5 bytes):
   - Tipo de mensaje (1 byte): Identifica el tipo de mwnsaje: `MSG_TYPE_BET = 1` (Mensaje de apuesta); `MSG_TYPE_ACK = 2` (Mensaje de confirmación)
   - Longitud del payload (4 bytes): Tamaño en bytes del contenido del mensaje (Big Endian)

2. Payload: Contenido serializado que contiene los datos de la apuesta en formato pipe-separated:
   ```
   nombre|apellido|documento|nacimiento|numero
   ```
   Ejemplo: `Tomas|Vainstein|00000000|2002-03-03|0000`
