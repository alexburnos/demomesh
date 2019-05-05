# Demomesh
Demomesh is a simple application that can be used to create demo mesh of services.

Demomesh helps to simulate microservices running independently, communicating
over network using JSON over HTTP and forming multi-tier application.

Each demomesh service has a frontend serving at root ("/" path) and a backend serving
at "/backend" path. On request to its frontend service will contact all its configured backends
to get information about them. Backends, in their turn, will query their backends and so on, until backend
with no backends configured is reached.

## How to run

Build binary with go build:
```
go build -o demomesh service/service.go
```

Run as many services as you need, configuring their backends as needed.

```
./demomesh -name "Frontend" -port 8080 -backends 127.0.0.1:9000,127.0.0.1:9001 &
./demomesh -name "Backend1" -port 9000 -backends 127.0.0.1:9001 &
./demomesh -name "Backend2" -port 9001 &
```

If you ran these commands on localhost, when you visit http://localhost:8080 in your browser
and see chain of requests that service "Frontend" received:

```
You landed at service "Frontend".
I process my requests through:
  Service name: Backend1
  Requested URL: http://127.0.0.1:9000/backend
  Hostname and port: rogueone:9000
  Backends:
      Service name: Backend2
      Requested URL: http://127.0.0.1:9001/backend
      Hostname and port: rogueone:9001
      I am a terminal service. No other is configured for me.
  Service name: Backend2
  Requested URL: http://127.0.0.1:9001/backend
  Hostname and port: rogueone:9001
  I am a terminal service. No other is configured for me.
```
