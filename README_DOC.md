# TP0: Documentación

## Ejercicio 1
En la resolución del ejercicio se creó un script de bash con nombre `generar-compose.sh` que permite generar un archivo Docker Compose con una cantidad configurable de clientes, siguiendo el formato `client1`, `client2`, `client3`, etc.

Parae ejecutar el script de bash se usa el comando:

`./generar-compose.sh docker-compose-dev.yaml 5`, pero primero hay que darle permisos de ejecución `chmod +x generar-compose.sh`.

El script verifica que se proporcionen los 2 parámetros: archivo de salida y cantidad de clientes. Luego se mantiene la configuración del server y client que ya se encuentran en el archivo docker-compose-dev original (`container_name: server`, `image: server:latest` y `entrypoint: python3 /main.py`) y se hace un bucle para generar la cantidad de clientes especificada por parámetro (también se mantiene la estructura del archivo original para los clientes). Para la configuración de red se usan los valores `testing_net` y `172.25.125.0/24`.

Para levantar los contenedores, se usa el comando `make docker-compose-up` finalmente y se pueden verificar que los clientes se comunican correctamente con el server verificando los logs con `make docker-compose-logs`. 
