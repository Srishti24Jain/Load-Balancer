package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Srishti24Jain/Load-Balancer/domain"
	"github.com/Srishti24Jain/Load-Balancer/usecase"
)

func main() {
	r := mux.NewRouter()

	//register endpoints
	r.HandleFunc("/urls/register", usecase.RegisterUrl).Methods("POST")

	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err.Error())
	}

	var cfg domain.RegisterUrls
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return
	}

	var servers []domain.Server
	for _, v := range cfg.Backends {
		servers = []domain.Server{usecase.NewSimpleServer(v.URL)}
	}

	server := usecase.NewLoadBalancer("8000", servers)

	//Proxy server
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		server.ServeProxy(rw, req)
	}

	// register a proxy handler to handle all requests
	r.HandleFunc("/proxy", handleRedirect)

	fmt.Printf("serving requests at 'localhost:%s'\n", server.Port)

	// start health checking
	go usecase.HealthCheck()

	err = http.ListenAndServe(":"+server.Port, r)
	if err != nil {
		return
	}
}
