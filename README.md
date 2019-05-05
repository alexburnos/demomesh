# Demomesh
Demomesh is a simple binary that can be used to create mesh of services.

Demomesh helps to simulate microservices running indepdendently, but communicating
over network using JSON over HTTP.

Each demomesh service has a frontend serving at root ("/" path) and a backend serving
at "/backend" path. Each service also has list of backends specified, which it will query
sequentially on each request to its frontend.

## How to run

Build binary with go build:
```
go build service.go
```

Run as many services as you need, configuring their backends as needed.

```
./service -name "Frontend" -port 8080 -backends 127.0.0.1:9000,127.0.0.1:9001 &
./service -name "Backend1" -port 9000 -backends 127.0.0.1:9001 &
./service -name "Backend2" -port 9001 &
```

If you ran these commands on localhost, when you visit http://localhost:8080 in your browser
and see chain of requests that service "Frontend" received:

```
You landed at service "Frontend".
I process my requests through:
    Service name: Backend1
    Url: http://127.0.0.1:9000/backend
    Hostname and port: rogueone:9000
    Backends:
        Service name: Backend2
        Url: http://127.0.0.1:9001/backend
        Hostname and port: rogueone:9001
        I am a terminal service. No other is configured for me.
    Service name: Backend2
    Url: http://127.0.0.1:9001/backend
    Hostname and port: rogueone:9001
    I am a terminal service. No other is configured for me.
```