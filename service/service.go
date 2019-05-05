package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	pb "github.com/alexburnos/demomesh/proto"
	"github.com/golang/protobuf/jsonpb"
)

// TCP port service will serve on.
var servingPort int

// Free form name of the service.
var serviceName string

// URL path for the service frontend.
var frontendPath = "/"

// URL path for the service backend.
var backendPath = "/backend"

// Serves HTML response to the "frontend" of the service.
func handleFrontend(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	reply := "<html><body>"
	reply += "<h2>You landed at service \"" + serviceName + "\".</h2>"

	if backends != nil {
		reply += "<h2>I process my requests through:</h2>"
		for _, backend := range backends {
			backendData := fetchBackendDataOverHTTP(backend)
			reply += backendReplyToHTML(backendData)
		}
		reply += "</body></html>"
	} else {
		reply += "<h2>I have no backends configured. Feeling lonely and useless.</h2>"
	}

	w.Write([]byte(reply))
}

// Populates BackendReply with service-specific parameters
func createBackendReply(backends []string) (reply pb.BackendReply) {
	params := &pb.BackendParams{Name: serviceName}
	params.Name = serviceName
	params.Hostname, _ = os.Hostname()
	params.Port = int32(servingPort)

	reply.Params = params

	// Resolution of service's upstream backends
	for _, backend := range backends {
		newBackend := fetchBackendDataOverHTTP(backend)
		reply.Backends = append(reply.Backends, &newBackend)
	}

	return
}

// Serves BackendReply as JSON over HTTP
func handleBackend(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "application/json")
	reply := createBackendReply(backends)

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
		out += "<li><u><font color=red>Backend Error</font></u>"
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
func fetchBackendDataOverHTTP(hostname string) (data pb.BackendReply) {
	fetchURL := "http://" + hostname + backendPath
	data.UrlRequested = fetchURL
	r, err := http.Get(fetchURL)
	if err != nil {
		backendError := createBackendReplyError(fetchURL, err)
		data.Error = &backendError
		return
	}

	jsonpb.Unmarshal(r.Body, &data)
	if err != nil {
		backendError := createBackendReplyError(fetchURL, err)
		data.Error = &backendError
		return
	}

	return
}

func main() {
	flag.StringVar(&serviceName, "name", "service", "A name of the service.")
	flag.IntVar(&servingPort, "port", 8080, "TCP port to serve service on.")
	backendsList := flag.String("backends", "", "Comma-separated list of HTTP backends in host:port format.")
	flag.Parse()

	var backends []string

	if *backendsList != "" {
		backends = strings.Split(*backendsList, ",")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleFrontend(w, r, backends)
	})
	http.HandleFunc("/backend", func(w http.ResponseWriter, r *http.Request) {
		handleBackend(w, r, backends)
	})

	fmt.Println("Starting serving service ", serviceName, " on port ", servingPort)
	if err := http.ListenAndServe(":"+strconv.Itoa(int(servingPort)), nil); err != nil {
		log.Fatal(err)
	}
}
