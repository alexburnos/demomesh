# Demomesh
Demomesh is a simple application that can be used to create demo mesh of services.

Demomesh helps to simulate microservices running independently, communicating
over network using JSON over HTTP or gRPC and forming multi-tier application.

Each demomesh service has two parts:
  * HTTP frontend that serves HTML with chain of replies of all backends of the services, as well as backends of the backends, etc.
  * HTTP or gRPC backend. It serves proto describing backend reply. Proto is served as JSON over HTTP or directly via gRPC
  (if -grpc flag is provided).

Both frontend and backend always receive results from their backends or an error before reply,
their backends do the same, etc, until request reaches service that does not have any backends configured.

## Usage
Build binary with go build:
```
go build -o demomesh service/service.go
```

Run as many services as you need, configuring their backends as needed.

```
./demomesh -name "Frontend" -fport 8080 -backends 127.0.0.1:9000,127.0.0.1:9001 &
./demomesh -name "Backend1" -fport=-1 -bport 9000 -backends 127.0.0.1:9001 &
./demomesh -name "Backend2" -fport=-1 -bport 9001 &
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
