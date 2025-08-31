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

## Ejercicio 3
En este ejercicio se creó un script de bash llamado `validar-echo-server.sh` que permite verificar el correcto funcionamiento del servidor echo usando netcat.


Se crea un contenedor Docker temporal con Busybox (que incluye netcat y no se instala en la máquina) y se conecta al servidor a través de la red Docker interna (`tp0_testing_net`). Después, se envía un mensaje de prueba y verifica que el servidor responda exactamente el mismo mensaje. Si el mensaje coincide, se imprime el resultado de éxito (success) y en el caso contrario el de error (fail).
También, se usa `--rm` para eliminar automáticamente el contenedor después de la prueba.


Para ejecutar el script se usa el comando `./validar-echo-server.sh`.

## Ejercicio 4
En este ejercicio se modificó tanto el servidor en Python como el cliente en Go para implementar un cierre graceful al recibir la señal de SIGTERM, lo cual implica que todos los recursos como file descriptors, sockets y conexiones se cierren correctamente antes de que la aplicación termine su ejecución, evitando pérdida de datos y recursos perdidos.

En el server:
- Los logs registran todos los pasos del cierre
- Se captura SIGTERM usando `signal.signal()`
- Se usa el flag `_running` para controlar el bucle principal
- El método `_cleanup()` cierra todos los recursos

En el cliente:
- Los logs registran todos los pasos del cierre
- Captura SIGTERM en una goroutine separada
- Se permite cancelar tanto en el bucle principal como en el sleep
- Método `cleanup()` que cierra conexiones y cancela contexto