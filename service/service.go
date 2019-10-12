package main

import (
	"bytes"
	"context"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	pb "github.com/alexburnos/demomesh/proto"
	"github.com/golang/protobuf/jsonpb"
	"go.opencensus.io/plugin/ochttp"
	"google.golang.org/grpc"
)

// TCP port service will serve its frontend on.
var frontendPort int

// TCP port service will serve its backend on.
var backendPort int

// Free form name of the service.
var serviceName string

// URL path for the service frontend.
var frontendPath = "/"

// URL path for the service backend.
var backendPath = "/backend"

// Port to use for GRPC serving. If not -1, then client connections will be over GRPC as well.
var enableGRPC bool

// List of hostnames in host:port format that are backends for this service
var backends []string

// A name of the HTTP header to propagate tracing information in.
var traceIDHeader = "x-request-id"

// Serves HTML response to the "frontend" of the service.
func handleFrontend(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	reply := "<html><body>"
	reply += "<h2>You landed at service \"" + serviceName + "\".</h2>"

	if backends != nil {
		reply += "<h2>I process my requests through:</h2>"
		for _, backend := range backends {
			var backendData pb.BackendReply
			if enableGRPC {
				backendData = fetchBackendDataOverGRPC(backend)
			} else {
				backendData = fetchBackendDataOverHTTP(backend, r.Context())
			}
			reply += backendReplyToHTML(backendData)
		}
		reply += "</body></html>"
	} else {
		reply += "<h2>I have no backends configured. Feeling lonely and useless.</h2>"
	}

	w.Write([]byte(reply))
}

// Populates BackendReply with service-specific parameters
func createBackendReply(backends []string, ctx context.Context) (reply pb.BackendReply) {
	params := &pb.BackendParams{Name: serviceName}
	params.Name = serviceName
	params.Hostname, _ = os.Hostname()
	params.Port = int32(frontendPort)

	reply.Params = params

	// Resolution of service's upstream backends
	for _, backend := range backends {
		var newBackend pb.BackendReply
		if enableGRPC {
			newBackend = fetchBackendDataOverGRPC(backend)
		} else {
			newBackend = fetchBackendDataOverHTTP(backend, ctx)
		}
		reply.Backends = append(reply.Backends, &newBackend)
	}

	return
}

// Serves BackendReply as JSON over HTTP
func handleBackendReplyOverHTTP(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "application/json")
	reply := createBackendReply(backends, r.Context())

	var m jsonpb.Marshaler
	b := new(bytes.Buffer)
	if err := m.Marshal(b, &reply); err != nil {
		log.Fatal("Could not marshal Struct to JSON.")
	}

	w.Write([]byte(b.String()))
}

// Converts BackendReply to HTML representation.
func backendReplyToHTML(backend pb.BackendReply) (out string) {
	out += "<ul>"
	if backend.Error != nil {
		out += "<li><u><font color=red>Backend Error</font>22</u>"
		out += "<li>Requested URL: " + backend.UrlRequested
		out += "<li>Error: " + backend.Error.ErrorString
		out += "</ul>"
		return
	}

	out += "<li><u>Service name:</u> " + backend.Params.Name
	out += "<li>Requested URL: " + backend.UrlRequested
	out += "<li>Hostname and port: " + backend.Params.Hostname + ":" + strconv.Itoa(int(backend.Params.Port))
	if backend.Backends != nil {
		out += "<li>Backends:"
		for _, innerBackend := range backend.Backends {
			out += backendReplyToHTML(*innerBackend)
		}
	} else {
		out += "<li>I am a terminal service. No other is configured for me."
	}
	out += "</ul>"

	return
}

// Helper to create a fake BackendData reply that serves as error message.
func createBackendReplyError(requestedURL string, err error) (backendError pb.BackendReplyError) {
	backendError.ErrorString = string(err.Error())
	backendError.IsError = true
	return
}

// Gets JSON from a backend via HTTP.
func fetchBackendDataOverHTTP(hostname string, ctx context.Context) (data pb.BackendReply) {
	fetchURL := "http://" + hostname + backendPath
	data.UrlRequested = fetchURL

	client := &http.Client{
		Transport: &ochttp.Transport{
			// Use Google Cloud propagation format.
			Propagation: &propagation.HTTPFormat{},
		},
	}
	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		backendError := createBackendReplyError(fetchURL, err)
		data.Error = &backendError
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}
	resp, err := client.Do(req)
	if err != nil {
		backendError := createBackendReplyError(fetchURL, err)
		data.Error = &backendError
		return
	}

	err = jsonpb.Unmarshal(resp.Body, &data)
	if err != nil {
		backendError := createBackendReplyError(fetchURL, err)
		data.Error = &backendError
		return
	}

	return
}

// Gets reply from backend over gRPC.
func fetchBackendDataOverGRPC(hostname string) (data pb.BackendReply) {
	debugHostname := "grpc://" + hostname
	conn, err := grpc.Dial(hostname, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewDemomeshServiceClient(conn)

	req := pb.BackendRequest{Id: 1}
	r, err := client.GetBackendInfo(context.Background(), &req)
	if err != nil {
		backendError := createBackendReplyError(debugHostname, err)
		data.Error = &backendError
		return
	}
	data = *r
	data.UrlRequested = debugHostname

	return
}

type demomeshServiceServer struct{}

func (s *demomeshServiceServer) GetBackendInfo(ctx context.Context, req *pb.BackendRequest) (*pb.BackendReply, error) {
	reply := createBackendReply(backends, nil)
	return &reply, nil
}

func main() {
	flag.StringVar(&serviceName, "name", "service", "A name of the service.")
	flag.IntVar(&frontendPort, "fport", 8080, "Port for HTTP frontend of the service.")
	flag.IntVar(&backendPort, "bport", 9000, "Port for HTTP or GRPC backend of the service.")
	flag.BoolVar(&enableGRPC, "grpc", false, "Uses GRPC instead of HTTP to serve and connect to other backends.")
	backendsList := flag.String("backends", "", "Comma-separated list of HTTP backends in host:port format.")
	flag.Parse()

	if frontendPort >= 0 {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleFrontend(w, r, backends)
		})

		go func() {
			fmt.Println("Starting HTTP frontend for service ", serviceName, " on port ", frontendPort)
			if err := http.ListenAndServe(":"+strconv.Itoa(int(frontendPort)), nil); err != nil {
				log.Fatal(err)
			}
		}()
	}

	if *backendsList != "" {
		backends = strings.Split(*backendsList, ",")
	}

	if enableGRPC {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", backendPort))
		if err != nil {
			log.Fatalf("Failed to listen for gRPC: %v", err)
		}
		fmt.Println("Starting gRPC backend for service ", serviceName, " on port ", backendPort)
		grpcServer := grpc.NewServer()
		s := demomeshServiceServer{}
		pb.RegisterDemomeshServiceServer(grpcServer, &s)
		grpcServer.Serve(l)
	} else {
		http.HandleFunc("/backend", func(w http.ResponseWriter, r *http.Request) {
			handleBackendReplyOverHTTP(w, r, backends)
		})
		fmt.Println("Starting HTTP backend for service ", serviceName, " on port ", backendPort)
		if err := http.ListenAndServe(":"+strconv.Itoa(int(backendPort)), nil); err != nil {
			log.Fatal(err)
		}
	}
}
