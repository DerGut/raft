package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/DerGut/raft/pkg/rpc"
)

type uris []string

var cluster uris
var cmd = flag.String("cmd", "", "The command to apply")

func init() {
	flag.Var(&cluster, "cluster", "URIs of the cluster members")
	flag.Parse()
}

func main() {
	if *cmd == "" {
		log.Fatal("Please provide a command with -cmd")
	}
	// choose random cluster member and connect
	member := chooseRandom(cluster)

	var res *rpc.ClientRequestResponse

	req := rpc.ClientRequestRequest{Cmds: []string{*cmd}}
	res, err := rpc.ClientRequest(req, member)
	if err != nil {
		log.Fatalln("Failed with", err)
	}
	if res.Success {
		log.Println("Success")
		return
	}
	if res.LeaderAddr != "" {
		log.Println("Got leader address, retrying")
		res, err = rpc.ClientRequest(req, res.LeaderAddr)
		if err != nil {
			log.Fatal("Failed with", err)
		}
	}
	log.Println("Response", *res)
}

func chooseRandom(cluster []string) string {
	return cluster[rand.Intn(len(cluster))]
}

func (u *uris) String() string {
	return fmt.Sprint(*u)
}

func (u *uris) Set(value string) error {
	if len(*u) > 0 {
		return errors.New("uris flag already set")
	}
	for _, v := range strings.Split(value, ",") {
		*u = append(*u, v)
	}
	return nil
}
