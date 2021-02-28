package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/DerGut/kv-store/cmd/raft/app"
	"github.com/DerGut/kv-store/raft"
	"github.com/DerGut/kv-store/raft/rpc"
	"github.com/DerGut/kv-store/raft/state"
	"github.com/DerGut/kv-store/server"
)

var debug = flag.Bool("debug", false, "Enable debug mode")
var address = flag.String("address", "127.0.0.1:3000", "Address and port to listen for connections")
var members app.URIs

func init() {
	flag.Var(&members, "members", "URIs of the cluster members")
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix(*address + " ")
}

func main() {
	err := app.SeedRand(*address)
	if err != nil {
		log.Fatal("Failed to seed rand with server address: ", err)
	}

	clusterRcvr := rpc.ClusterReceiver{
		AppendEntriesRequests:  make(chan rpc.AppendEntriesRequest, 1),
		AppendEntriesResponses: make(chan rpc.AppendEntriesResponse, 1),
		RequestVoteRequests:    make(chan rpc.RequestVoteRequest, 1),
		RequestVoteResponses:   make(chan rpc.RequestVoteResponse, 1),
	}
	clientRcvr := rpc.ClientReceiver{
		ClientRequests:  make(chan rpc.ClientRequestRequest, 1),
		ClientResponses: make(chan rpc.ClientRequestResponse, 1),
	}

	l, err := rpc.RegisterReceivers(&clusterRcvr, &clientRcvr, *address)
	if err != nil {
		log.Fatalln("Failed to register RPC receivers", err)
	}

	go http.Serve(l, nil)

	r := raft.Raft{
		ClusterOptions:  server.ClusterOptions{Address: *address, Members: members},
		State:           state.NewState(),
		ClusterReceiver: clusterRcvr,
		ClientReceiver:  clientRcvr,
	}

	r.Run()
}
