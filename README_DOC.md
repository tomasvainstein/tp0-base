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

Se agregó la función `sendBetBatch` que permite enviar múltiples apuestas en un solo mensaje. La lectura de apuestas desde archivos CSV procesa cada línea de manera incremental y valida que cada apuesta cumpla con los campos requeridos. Una vez cargadas todas las apuestas en memoria, se divide el conjunto en lotes según el parámetro `BatchMaxAmount` definido en el config del cliente, permitiendo ajustar el tamaño de cada envío.

Del lado del servidor, se implementó la función `parse_bet_batch_payload` para deserializar los lotes de apuestas recibidos y procesarlos de forma agrupada. Además, `send_ack` envía una confirmación de éxito o un mensaje de error detallando lo que se hizo con el mensaje despues de procesar cada batch.

La configuración del tamaño de los lotes se define en config bajo la clave batch.maxAmount, con un valor por defecto de 160 apuestas por lote. Este umbral se calculó para ajustar un límite aproximado de 8 kB por mensaje, asumiendo un tamaño medio de 50 bytes por apuesta.

El protocolo de comunicación sigue estos pasos: primero el cliente carga y valida los datos CSV, luego crea los lotes y los envía en conexiones TCP separadas. Tras cada envío, espera el ACK del servidor antes de continuar con el siguiente lote. En sentido contrario, el servidor recibe el batch, valida la estructura, persiste las apuestas usando `store_bets()`, y finalmente envía la confirmación de éxito o un reporte de error, logeando el mismo por consola también.

## Ejercicio 7
En este ejercicio se implementó el sistema de loteria con sorteo y consulta de ganadores, donde los clientes notifican al servidor cuando terminan de enviar todas sus apuestas y el servidor realiza el sorteo solo cuando todas las agencias han finalizado.

El flujo del sistema funciona así:

- Envío de apuestas: Los clientes envían sus apuestas por lotes como en el ejercicio anterior, pero ahora cada apuesta incluye el id de la agencia en el payload.

- Notificación de finalización: Una vez que cada cliente termina de enviar todas sus apuestas, envía una notificación de finalización al servidor indicando que ha completado su proceso.

- Sorteo dinámico: El servidor mantiene un registro de todas las agencias que han enviado apuestas y espera a que todas notifiquen su finalización antes de realizar el sorteo.

- Consulta de Ganadores: Después del sorteo, todos los clientes reciben automáticamente la cantidad de ganadores que tienen, sin necesidad de realizar consultas adicionales.

Protocolo de comunicación:

Se agregaron tres nuevos tipos de mensaje al protocolo existente:

- `MSG_TYPE_FINISH_NOTIFICATION = 3`: Mensaje de notificación de finalización
- `MSG_TYPE_WINNER_QUERY = 4`: Mensaje de consulta de ganadores  
- `MSG_TYPE_WINNER_RESPONSE = 5`: Mensaje de respuesta con cantidad de ganadores

Gestión de estado del servidor:

El servidor mantiene tres estructuras de estado principales:

- `_agencies_with_bets`: Conjunto de ids de agencias que han enviado al menos una apuesta
- `_finished_agencies`: Conjunto de ids de agencias que han notificado finalización
- `_waiting_clients`: Diccionario que mantiene las conexiones de clientes esperando recibir ganadores

Sincronización y Sorteo:

El sorteo se realiza únicamente cuando se cumple la condición: `len(_finished_agencies) == len(_agencies_with_bets)`. Esto se hace para que todas las agencias que participaron en el proceso hayan notificado su finalización antes de proceder con el sorteo.
Una vez realizado el sorteo, el servidor calcula los ganadores por agencia y envía la información a todos los clientes que están esperando, manteniendo sus conexiones abiertas hasta recibir la respuesta.