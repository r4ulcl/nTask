# nTask

nTask is a program for distributing tasks (any command or program) among different computers using API communications, both for managing the Manager and for the workers. The main idea is to be able to launch task requests from any client to the manager for it to handle them. The manager sends these tasks in order to the different available workers, receiving a request from the worker with the execution result. Once this is done, it is stored in the database and optionally can be sent to a URL/API to manage the output in another program or API.

The manager uses a MySQL database to store all the information, storing both the information of each worker and all the task information. The manager also has a public API that is accessed with an authentication token.

The idea is to connect another API, a Telegram bot or a simple bash script to this API to process task. 

## Features

- Manager API to send tasks.
- Multiples workers.
- MySQL database for save all tasks information.
- Task modules configured in worker.conf JSON. 
- Same binary for manager and worker.
- Multiple commands in a task, to execute sequential in a worker.
- Send file in a task and save in a custom path.
- Whitelist in workers to only access from manager.
- Docker and docker compose.
- Multiples users in manager (using oauth token).
- Each worker with a different token.
- TLS in manager and between Manager and Workers, verifying the CA. 
- Ability to configure one VPS and clone it using the different hostnames as ids. 
- Compatible with dynamic IPs in workers.
- Callback option after task executed.
- Output to file.
- Swagger documentation.
- Swagger web (optional).

## Installation

### Docker

Both images use the same binary, but the manager uses a scratch image just to run the binary and the worker uses a kali-linux image to install tools and dependencies easier. 

``` bash
docker pull r4ulcl/nTask-manager
docker pull r4ulcl/nTask-worker
```

### Manual install

To use the nTask manager, you will need Go installed on your machine. You can download and install Go from the official website: [https://golang.org](https://golang.org).

Once Go is installed, you can clone the repository and build the manager:

```
$ git clone https://github.com/r4ulcl/nTask.git
$ cd nTask
$ go build
```

## Configuration

### SSL (optional)

You can use any certificate for the manager and the worker. If you want to use a self signed certificate you can execute the following code, by default the manager and workers only check the certificate, not the IP or domain. If you want to check fully the certificate edit the script with the correct fields and use the flag `-verifyAltName`.

``` bash
bash generateCert.sh
```

### Manager

The manager requires a configuration file named `manager.conf` to be present in the same directory as the executable. The configuration file should be in JSON format and contain the following fields:

  ```json
  {
  "users": {
    "user1" : "WLJ2xVQZ5TXVw4qEznZDnmEEV",
    "user2" : "WLJ2xVQZ5TXVw4qEznZDnmEE2",
    "user3" : "WLJ2xVQZ5TXVw4qEznZDnmEE3"
  },
    "workers": {
      "workers" : "IeH0vpYFz2Yol6RdLvYZz62TFMv5FF"
  },
  "port": "8080",
  "dbUsername" : "your_username",
  "dbPassword" : "your_password",
  "dbHost" : "db",
  "dbPort" : "3306",
  "dbDatabase" : "manager",
  "callbackURL" : "",
  "callbackToken" : "",
  "diskPath": "./output"
}
  ```

- `oauthToken`: OauthToken for user in the manager API.
- `oauthTokenWorkers`: OauthToken for the workers. this way the worker token only can do worker related requests. 
- `Port`: The port on which the manager should listen for incoming connections.
- `DBUsername`: The username for the database connection.
- `DBPassword`: The password for the database connection.
- `DBHost`: The hostname of the database server.
- `DBPort`: The port number of the database server.
- `DBDatabase`: The name of the database to use.
- `callbackURL`: (optional) CallbackURL to send a POST request with the Task when done.
- `callbackToken`: (optional) CallbackToken for the OauthToken in the Callback request. 
- `diskPath`: (optional) Folder to save the tasks output

### Worker

Create a configuration file `workerouter.conf` with the following structure:

  ```json
{
  "name": "",
  "iddleThreads": 1,
  "managerIP" : "nTask_manager",
  "managerPort" : "8080",
  "managerOauthToken": "IeH0vpYFz2Yol6RdLvYZz62TFMv5FF",
  "oauthToken": "",
  "port": "8081",
  "modules": {
    "sleep": "/usr/bin/sleep",
    "curl": "/usr/bin/curl",
    "module1": "python3 ./worker/modules/module1.py",
    "exec": ""
  }
}

   ```

   - `name`: (optional) The name of the worker. If not provided, the hostname will be used.
   - `iddleThreads`: Number of threads in the worker (default 1)
   - `managerIP`: Manager IP or domain
   - `managerPort`: manager port
   - `managerOauthToken`: Manager configured OauthToken for workers
   - `oauthToken`: (optional) OauthToken for the worker. If not provided, the worker will set a random one on start. 
   - `port`: The port number on which the worker should listen for incoming requests.
   - `modules`: A map of module names to executable commands.
   
Each worker uses to identify itself as unique to the manager the name and the ip:port, so if the name is left blank and the IP and port of each client is different, the same VPS can be cloned indefinitely if each VPS has a different hostname. 

## Usage manager

I recommend the following configuration:
- Manager:
  - Execute the manager in a docker compose in the manager sever.
- Worker:
  - Create a new Dockerfile installing the needed tools in the docker for the workers.
  - Create a VPS, install all the tools and nTask and execute it there.
  - If you want to execute external tools in docker you cant share the docker.sock with this docker and execute any docker from the nTask docker. 

### Docker compose

Once the configuration files have been modified. To run nTask in manager mode the easiest way is to run the docker compose manager as follows. 

``` bash
docker compose -f docker-compose-manager.yml up -d 
```

### Binary 

To start the manager, run the executable:

```
$ ./nTask manager
```

The manager will read the configuration file, connect to the database, and start listening for incoming connections on the specified port.

## Usage worker

### Docker compose

Once the manager is up, we can run the following docker compose on each worker instance
``` bash
docker compose -f docker-compose-worker.yml up -d 
```

### Binary 

```
$ ./nTask worker
```

### Custom Dockerfile

Edit the `./worker/Dockerfile` file adding the needed tools for the modules. You can also modify the docker image, the default one is Kali. 


## Flags

 - `--manager`: Run nTask as manager
 - `--worker`: Run nTask as worker
 - `--swagger`: Start the swager endpoint (/swagger)
 - `--verbose`: Set verbose mode
 - `--configFile`: Path to the config file for manager and worker


## API Endpoints manager

The nTask manager exposes the following API endpoints for the user/manager:

- `GET /task`: Get information about all tasks.
- `POST /task`: Add a new task.
- `DELETE /task/{ID}`: Delete a task with the specified ID.
- `GET /task/{ID}`: Get the status of a task with the specified ID.

API endpoint for workers:
- `GET /worker`: Get information about all workers.
- `POST /worker`: Add a new worker.
- `DELETE /worker/{NAME}`: Delete a worker with the specified name.
- `GET /worker/{NAME}`: Get the status of a worker with the specified name.
- `POST /callback`: Receive callback information from a task.

The API endpoints can be accessed using a REST client such as cURL or Postman.

## Swagger Documentation

### Generating Swagger docs
``` bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

The nTask manager also provides Swagger documentation for its API. You can access the Swagger UI at `/swagger/` and the Swagger JSON at `/docs/swagger.json`.

## Diagram

![Diagram](./resources/nTask-diagram.png)


## TODO
- Add cloud instances
    - DigitalOcean

## Author

- Raúl Calvo Laorden (@r4ulcl)

## Support this project

### Buymeacoffee

[<img src="https://cdn.buymeacoffee.com/buttons/v2/default-green.png">](https://www.buymeacoffee.com/r4ulcl)

## License

[GNU General Public License v3.0](https://github.com/r4ulcl/nTask/blob/master/LICENSE)