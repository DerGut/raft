package main

import (
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/DerGut/kv-store/server"
)

type uris []string

var debug = flag.Bool("debug", false, "Enable debug mode")
var address = flag.String("address", "127.0.0.1:3000", "Address and port to listen for connections")
var members uris

func init() {
	flag.Var(&members, "members", "URIs of the cluster members")
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix(*address)
}

func main() {
	err := seedRand(*address)
	if err != nil {
		log.Fatal("Failed to seed rand with server address: ", err)
	}

	s := server.NewServer(*address, members)
	l := registerServer(s)
	go http.Serve(l, nil)

	if err := s.Run(*debug); err != nil {
		log.Fatal("Failed to run server: ", err)
	}
}

func seedRand(address string) error {
	hashAlg := fnv.New64()
	b := []byte(address)
	if _, err := hashAlg.Write(b); err != nil {
		return err
	}
	hashSum := hashAlg.Sum64()

	rand.Seed(int64(hashSum))
	return nil
}

func registerServer(s *server.Server) net.Listener {
	if err := rpc.Register(s); err != nil {
		log.Fatalln("Failed to register:", err)
	}
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", s.MemberID)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	return l
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
