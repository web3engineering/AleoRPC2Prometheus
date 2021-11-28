package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	currentBlock = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_block",
		Help: "Latest Block known by node.",
	})
	currentStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "status",
		Help: "Status: PEERING=1, SYNCING=2, READY=3, MINING=4, UNKNOWN=5",
	})
)

var (
	status_request = []byte(`{"jsonrpc": "2.0", "id":"documentation", "method": "getnodestate", "params": []}`)
)

var (
	rcp_address = flag.String("rpc-address", "http://127.0.0.1:3032", "The address of RPC server.")
	listen_port = flag.String("port", ":9090", "The address to listen for metrics server.")
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(currentBlock)
	prometheus.MustRegister(currentStatus)
}

type NodestateResult struct {
	Status            string `json:"status"`
	LatestBlockHeight int    `json:"latest_block_height"`
}

type RPCNodestateResponse struct {
	Result NodestateResult `json:"result"`
}

func main() {
	flag.Parse()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Post(*rcp_address, "application/json", bytes.NewBuffer(status_request))
		if err != nil {
			log.Fatal("Error getting response. ", err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Error reading response. ", err)
		}

		var rpc_result RPCNodestateResponse
		err = json.Unmarshal(body, &rpc_result)
		if err != nil {
			log.Fatal("Unable to unmarshall")
		}

		switch rpc_result.Result.Status {
		case "Peering":
			currentStatus.Set(1)
		case "Syncing":
			currentStatus.Set(2)
		case "Ready":
			currentStatus.Set(3)
		case "Mining":
			currentStatus.Set(4)
		default:
			currentStatus.Set(0)
		}

		currentBlock.Set(float64(rpc_result.Result.LatestBlockHeight))
		next := promhttp.HandlerFor(
			prometheus.DefaultGatherer, promhttp.HandlerOpts{})
		next.ServeHTTP(w, r)
	})
	fmt.Printf("Starting server at port %s\n", *listen_port)
	if err := http.ListenAndServe(*listen_port, nil); err != nil {
		log.Fatal(err)
	}
}
