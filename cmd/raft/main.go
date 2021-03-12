package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/DerGut/raft/cmd/raft/app"
	"github.com/DerGut/raft/pkg/raft"
	"github.com/DerGut/raft/pkg/rpc"
	"github.com/DerGut/raft/pkg/state"
	"github.com/DerGut/raft/server"
)

var debug = flag.Bool("debug", false, "Enable debug mode")
var address = flag.String("address", "127.0.0.1:3000", "Address and port to listen for connections")
var statePath = flag.String("state", "/etc/raft", "Directory path where to store persistent state")
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

	m := app.StateMachine{}
	s, err := state.NewState(&m, *statePath)
	if err != nil {
		log.Fatalln("Failed to create state", err)
	}
	r := raft.Raft{
		ClusterOptions:  server.ClusterOptions{Address: *address, Members: members},
		State:           s,
		ClusterReceiver: clusterRcvr,
		ClientReceiver:  clientRcvr,
	}

	r.Run()
}
