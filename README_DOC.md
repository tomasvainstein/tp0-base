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


## Ejercicio 6
En este ejercicio se implementó el procesamiento por batches para el sistema de apuestas, permitiendo que se envíen múltiples apuestas en una sola comunicación con el servidor.


Donde cada apuesta mantiene el formato csv en pipe-separated: `nombre|apellido|documento|nacimiento|numero`


### Ejemplo:
```
3
Juan|Pérez|12345678|1990-01-01|1234
María|García|87654321|1985-05-15|5678
Carlos|López|11223344|1992-12-25|9012
```

Se agregó la función `sendBetBatch` que permite enviar múltiples apuestas en un solo mensaje. La lectura de apuestas desde archivos CSV se realiza de manera streaming utilizando `encoding/csv`, procesando cada línea incrementalmente sin cargar todo el archivo en memoria. Cada apuesta se valida individualmente y se agrupa en lotes según el parámetro `BatchMaxAmount` definido en el config del cliente, permitiendo ajustar el tamaño de cada envío.

Del lado del servidor, se implementó la función `parse_bet_batch_payload` para deserializar los lotes de apuestas recibidos y procesarlos de forma agrupada. Además, `send_ack` envía una confirmación de éxito o un mensaje de error detallando lo que se hizo con el mensaje despues de procesar cada batch.

La configuración del tamaño de los lotes se define en config bajo la clave batch.maxAmount, con un valor por defecto de 160 apuestas por lote. Este umbral se calculó para ajustar un límite aproximado de 8 kB por mensaje, asumiendo un tamaño medio de 50 bytes por apuesta.

El protocolo de comunicación sigue estos pasos: primero el cliente procesa el archivo csv, validando cada apuesta individualmente y agrupándolas en lotes de tamaño configurable. Los lotes se envían en conexiones TCP separadas tan pronto como se completan. Tras cada envío, espera el ACK del servidor antes de continuar con el siguiente lote. En sentido contrario, el servidor recibe el batch, valida la estructura, persiste las apuestas usando `store_bets()`, y finalmente envía la confirmación de éxito o un reporte de error, logeando el mismo por consola también.