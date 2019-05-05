package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// TCP port service will serve on.
var servingPort int

// Free form name of the service.
var serviceName string

// URL path for the service frontend.
var frontendPath = "/"

// URL path for the service backend.
var backendPath = "/backend"

// Backend service reply message that holds all metadata
// about service that served it.
type BackendData struct {
	Name      string
	Port      int
	Hostname  string
	Url       string
	ErrStatus string
	Backends  []BackendData
}

// Serves HTML response to the "frontend" of the service.
func handleFrontend(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	reply := "<html><body>"
	reply += "<h2>You landed at service \"" + serviceName + "\".</h2>"

	if backends != nil {
		reply += "<h2>I process my requests through:</h2>"
		for _, backend := range backends {
			backendData := fetchBackendDataOverHTTP(backend)
			reply += backendDataToHTML(backendData)
		}
		reply += "</body></html>"
	} else {
		reply += "<h2>I have no backends configured. Feeling lonely and useless.</h2>"
	}

	w.Write([]byte(reply))
}

// Serves BackendData JSON response to the backend of the service.
func handleBackend(w http.ResponseWriter, r *http.Request, backends []string) {
	w.Header().Set("Content-Type", "application/json")
	reply := BackendData{}
	reply.Name = serviceName
	reply.Hostname, _ = os.Hostname()
	reply.Port = servingPort

	for _, backend := range backends {
		reply.Backends = append(reply.Backends, fetchBackendDataOverHTTP(backend))
	}

	jsonReply, err := json.Marshal(&reply)
	if err != nil {
		log.Fatal("Could not marshal Struct to JSON.")
	}

	w.Write([]byte(jsonReply))
}

// Converts BackendData to HTML representation.
func backendDataToHTML(backend BackendData) (out string) {
	out += "<ul>"
	if backend.ErrStatus != "" {
		out += "<li>"
		out += backend.Name + " backend Error: " + backend.ErrStatus
		out += "</ul>"
		return
	}

	out += "<li><u>Service name:</u> " + backend.Name
	out += "<li>Url: " + backend.Url
	out += "<li>Hostname and port: " + backend.Hostname + ":" + strconv.Itoa(backend.Port)
	if backend.Backends != nil {
		out += "<li>Backends:"
		for _, innerBackend := range backend.Backends {
			out += backendDataToHTML(innerBackend)
		}
	} else {
		out += "<li>I am a terminal service. No other is configured for me."
	}
	out += "</ul>"

	return
}

// Helper to create a fake BackendData reply that serves as error message.
func CreateBackendWithError(name string, err error) (errBackend BackendData) {
	errBackend.Name = name
	errBackend.ErrStatus = string(err.Error())
	return
}

// Gets JSON from a backend via HTTP.
func fetchBackendDataOverHTTP(hostname string) (data BackendData) {
	fetchUrl := "http://" + hostname + backendPath
	r, err := http.Get(fetchUrl)
	if err != nil {
		return CreateBackendWithError(hostname, err)
	}
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return CreateBackendWithError(hostname, err)
	}

	if err = json.Unmarshal([]byte(content), &data); err != nil {
		return CreateBackendWithError(hostname, err)
	}

	data.Url = fetchUrl

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
	if err := http.ListenAndServe(":"+strconv.Itoa(servingPort), nil); err != nil {
		log.Fatal(err)
	}
}
