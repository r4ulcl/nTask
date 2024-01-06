<p align="center">
  <img src="resources/nTask_small.png" style="width: 70%; height: 70%"/>>
</p>

<p align="center">
   <a href="https://github.com/r4ulcl/nTask/releases">
    <img src="https://img.shields.io/github/v/release/r4ulcl/nTask" alt="GitHub releases">
  </a>
  <a href="https://github.com/r4ulcl/nTask/stargazers">
    <img src="https://img.shields.io/github/stars/r4ulcl/nTask.svg" alt="GitHub stars">
  </a>
  <a href="https://github.com/r4ulcl/nTask/network">
    <img src="https://img.shields.io/github/forks/r4ulcl/nTask.svg" alt="GitHub forks">
  </a>
  <a href="https://github.com/r4ulcl/nTask/issues">
    <img src="https://img.shields.io/github/issues/r4ulcl/nTask.svg" alt="GitHub issues">
  </a>
  <a href="https://www.codefactor.io/repository/github/r4ulcl/nTask">
    <img src="https://www.codefactor.io/repository/github/r4ulcl/nTask/badge" alt="CodeFactor" />
  </a>
    <a href="https://github.com/r4ulcl/nTask">
    <img src="https://tokei.rs/b1/github/r4ulcl/nTask" alt="LoC" />
  </a>
  <a href="https://github.com/r4ulcl/nTask/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/r4ulcl/nTask.svg" alt="GitHub license">
  </a>

  <br>
  <a href="https://hub.docker.com/r/r4ulcl/nTask">
    <img src="https://github.com/r4ulcl/nTask/actions/workflows/docker-image.yml/badge.svg" alt="Docker Image">
  </a>
    <a href="https://hub.docker.com/r/r4ulcl/nTask/tags">
    <img src="https://github.com/r4ulcl/nTask/actions/workflows/docker-image-dev.yml/badge.svg" alt="Docker Image dev">
  </a>
  
  </p>

# nTask

nTask is a program that allows you to distribute tasks (any command or program) among different computers using API communications and WebSockets. The main idea is to be able to launch task requests from any client to the manager for it to handle them. The manager sends these tasks in order to the different available workers, receiving a request from the worker with the execution result. Once this is done, it is stored in the database and optionally can be sent to a URL/API to manage the output in another program or API.

The manager uses a MySQL database to store all the information, storing both the information of each worker and all the task information. The manager also has a public API that is accessed with an authentication token.

The idea is to connect another API, a Telegram bot or a simple bash script to this API to process tasks. 

## Features

- Worker connects to Manager using WebSockets.
- Workers connects Manager using WebSockets.
- Support for multiple workers.
- MySQL database to store task information.
- Configuration of task modules in worker.conf JSON.
- Same binary for manager and worker.
- Support for multiple commands in a task, allowing sequential execution in a worker.
- Ability to send files as part of a task and save them to a custom path.
- Optional SSH tunneling to securely send the manager API port to clients without exposing it.
- Docker and Docker Compose support.
- Support for multiple users in the manager using OAuth tokens.
- Works on Linux and Windows. 
- Each worker can have a unique token for authentication.
- Each worker can execute a configurable number of tasks in parallel.
- TLS support for secure communication between manager and workers, with certificate verification.
- Ability to configure one VPS and clone it using different hostnames as IDs.
- Compatible with dynamic IPs in workers (and manager if SSH tunneling is used).
- Callback option after task execution.
- Output logging to file.
- Swagger documentation.
- Optional Swagger web interface.

## Installation

### Docker

Both images use the same binary, but the manager uses a scratch image just to run the binary and the worker uses a kali-linux image to install tools and dependencies easier. 

You can pull these images from Docker Hub:

``` bash
docker pull r4ulcl/nTask-manager
docker pull r4ulcl/nTask-worker
```

### Manual Installation

To install nTask manually, you need to have Go installed on your machine. You can download and install Go from the official website: [https://golang.org](https://golang.org).

Once Go is installed, you can clone the repository and build the manager:

```bash
go install github.com/r4ulcl/nTask
```

Or, you can clone the repository and build the manager using the following commands:

```bash
git clone https://github.com/r4ulcl/nTask.git
cd nTask
go build
```

## Configuration

### SSL (optional)

You can use any certificate for the manager and the worker. If you want to use a self signed certificate you can execute the following code, by default the manager and workers only check the certificate, not the IP or domain. If you want to check fully the certificate edit the script with the correct fields and use the flag `-verifyAltName`.

``` bash
bash generateCert.sh
```

Set the certificate folder in the `certFolder` variable in the `manager.conf` config file. 

### Manager

The manager requires a configuration file named `manager.conf` to be present in the same directory as the executable. The configuration file should be in JSON format and contain the following fields:

  ```json
  {
  "users": {
    "user1": "WLJ2xVQZ5TXVw4qEznZDnmEEV",
    "user2": "WLJ2xVQZ5TXVw4qEznZDnmEE2",
    "user3": "WLJ2xVQZ5TXVw4qEznZDnmEE3"
  },
  "workers": {
      "workers": "IeH0vpYFz2Yol6RdLvYZz62TFMv5FF"
  },
  "statusCheckSeconds": 10,
  "StatusCheckDown": 360,
  "port": "8080",
  "dbUsername": "your_username",
  "dbPassword": "your_password",
  "dbHost": "db",
  "dbPort": "3306",
  "dbDatabase": "manager",
  "diskPath": "",
  "certFolder": "./certs/manager/"
}
```

- `users`: A map of user names and their corresponding OAuth tokens for authentication.
- `workers`: A map of worker names and their corresponding tokens for authentication.
- `statusCheckSeconds`: The interval in seconds between status check requests from the manager to the workers.
- `StatusCheckDown`: The number of seconds after which a worker is marked as down if the status check request fails.
- `port`: The port on which the manager should listen for incoming connections.
- `dbUsername`: The username for the database connection.
- `dbPassword`: The password for the database connection.
- `dbHost`: The hostname of the database server.
- `dbPort`: The port number of the database server.
- `dbDatabase`: The name of the database to use.
- `diskPath`: (optional) The folder path where task outputs should be saved.
- `certFolder`: The folder path where SSL certificates for the manager should be stored.

### Worker

The worker requires a configuration file named `workerouter.conf` to be present in the same directory as the executable. The configuration file should be in JSON format and contain the following fields:

```json
{
  "name": "",
  "iddleThreads": 2,
  "managerIP": "127.0.0.1",
  "managerPort": "8080",
  "managerOauthToken": "IeH0vpYFz2Yol6RdLvYZz62TFMv5FF",
  "CA": "./certs/ca-cert.pem",
  "insecureModules": true,
  "modules": {
    "sleep": "/usr/bin/sleep",
    "curl": "/usr/bin/curl",
    "echo": "/usr/bin/echo",
    "cat": "/usr/bin/cat",
    "grep": "/usr/bin/grep",
    "nmap": "nmap",
    "nmapIPs": "bash ./worker/modules/nmapIPs.sh",
    "exec": ""
  }
}
```

- `name`: (optional) The name of the worker. If not provided, the hostname will be used.
- `iddleThreads`: The number of idle threads in the worker (default: 5).
- `managerIP`: The IP address or domain name of the manager.
- `managerPort`: The port on which the manager is listening.
- `managerOauthToken`: The OAuth token for authentication with the manager.
- `port`: The port number on which the worker should listen for incoming requests.
- `CA`: The path to the CA certificate used for TLS communication with the manager.
- `insecureModules`: This flag determines whether the worker allows the execution of insecure modules with special characters like `;` or `|`.
- `modules`: A map of module names to executable commands.

Note: The `exec` module and the `insecureModules` flag allow remote execution of arbitrary commands on the worker. Use them with caution.
   
Each worker uses a unique name and IP:port combination to identify itself to the manager. If the name is left blank and the IP and port are different for each client, the same VPS can be cloned indefinitely as long as each VPS has a different hostname.

## Usage manager

An usage example can be found here: https://r4ulcl.com/posts

I recommend the following configuration:
- Manager:
  - Execute the manager in a docker compose in the manager sever.
- Worker:
  - Create a new Dockerfile installing the needed tools in the docker for the workers.
  - Create a VPS, install all the tools and nTask and execute it there.
  - If you want to execute external tools in docker you cant share the docker.sock with this docker and execute any docker from the nTask docker. 

### Manager flags 

  - `-c`, `--configFile` string      Path to the config file (default: manager.conf)
  - `-f`, `--configSSHFile` string   Path to the config SSH file (default empty)
  - `-h`, `--help`                   help for manager


### Docker compose

Once the configuration files have been modified. To run nTask in manager mode the easiest way is to run the docker compose manager as follows. 

``` bash
docker compose up manager -d
```

### Binary 

To start the manager, run the executable:

``` bash
$ ./nTask manager
```

The manager will read the configuration file, connect to the database, and start listening for incoming connections on the specified port.

## Usage worker

### Worker flags 

  - `-c`, `--configFile` string      Path to the config file (default: worker.conf)
  - `-h`, `--help`                   help for manager

### Docker compose

Once the manager is up, we can run the following docker compose on each worker instance

``` bash
docker compose up worker -d
```

### Binary 

``` bash
$ ./nTask worker
```

### Custom Dockerfile

Edit the `./worker/Dockerfile` file adding the needed tools for the modules. You can also modify the docker image, the default one is Kali. 


## Secure

To ensure the security of the nTask Manager, we recommend implementing the following measures:
- Use a legitimate TLS certificate to secure communication between the manager and the workers.
- Change the default port to a high port.
- Filter with `iptables` the input to allow only the IPs of the workers.
- Create an SSH tunnel to prevent the API port from being exposed on the internet.

### Use SSH tunnels

Using SSH tunnels is a recommended method to enhance the security of the nTask Manager. By configuring SSH tunnels, the manager can send the port to each worker without exposing the API to the internet.

#### SSH config file

To connect a SSH server automatcally from nTask you need a private certificate with access to the server and to confiure a configSSHFile:

``` bash
{
  "ipPort": {
    "<IP1>" : "22",
    "<IP2>" : "22",
    "<IP3>" : "22"
  },
  "username": "root",
  "privateKeyPath": "~/.ssh/ssh_key",
  "privateKeyPassword": ""
}
```

- `ipPort`: List of ip and port combination to connect to with SSH. 
- `username`: User to access via SSH.
- `privateKeyPath`: Path to the SSH private key.
- `privateKeyPassword`: (Optional) Password for the private key.

#### Manually

Alternatively, you can establish an SSH tunnel manually by following these steps:

```bash
ssh -L local_port:remote_server:remote_port -R remote_port:localhost:local_port user@remote_server
```

 Replace `local_port` with the port number on the manager machine, `remote_server` with the IP address or hostname of the worker machine, `remote_port` with the port number on the worker machine, and `user` with the SSH user.

   This command establishes a tunnel between the manager and the worker, allowing secure communication without exposing the API to the internet.

## Global flags

The nTask Manager supports the following global flags:

- `--swagger`: Enables the Swagger endpoint (/swagger) to access API documentation and interact with the API using its UI.
- `--debug`: Sets the manager in debug mode, providing additional logging and diagnostics information.

## API Endpoints

The nTask Manager exposes the following API endpoints:

### Manager Endpoints

- `GET /task`: Retrieves information about all tasks.
- `POST /task`: Adds a new task.
- `DELETE /task/{ID}`: Deletes a task with the specified ID.
- `GET /task/{ID}`: Retrieves the status of a task with the specified ID.

### Worker Endpoints

- `GET /worker`: Retrieves information about all workers.
- `POST /worker`: Adds a new worker.
- `DELETE /worker/{NAME}`: Deletes a worker with the specified name.
- `GET /worker/{NAME}`: Retrieves the status of a worker with the specified name.

You can access these API endpoints using a REST client such as cURL or Postman.

## Swagger Documentation

The nTask Manager provides Swagger documentation for its API, which allows for easier understanding and testing of the available endpoints.

### Generating Swagger docs

To generate the Swagger documentation, follow these steps:

1. Install the latest version of the `swag` command-line tool by running the following command:

   ``` bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. Initialize the Swagger docs by running the following command:

   ``` bash
   swag init
   ```

   This command generates the Swagger JSON and the necessary files for the Swagger UI.
## Diagram

![nTask Diagram](./resources/nTask-diagram.png)

The diagram above illustrates the architecture of the nTask Manager and its interactions with the workers.

## TODO
- Code tests
- DigitalOcean API

## Author

- Raúl Calvo Laorden (@r4ulcl)

## Support this project

### Buymeacoffee

[<img src="https://cdn.buymeacoffee.com/buttons/v2/default-green.png">](https://www.buymeacoffee.com/r4ulcl)

## License

[GNU General Public License v3.0](https://github.com/r4ulcl/nTask/blob/master/LICENSE)
