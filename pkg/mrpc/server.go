package mrpc

import (
	"log"
	"net"
	"net/rpc"
)

func ListenAndServer(addr string, serves ...interface{}) {
	var err error
	for _, v := range serves {
		if err = rpc.Register(v); err != nil {
			log.Fatalln("Add service failed, " + err.Error())
		}
	}
	l, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatalf("Listen %s failed, %s", addr, err.Error())
	}
	defer func() {
		if err = l.Close(); err != nil {
			log.Fatalln("Listen " + addr + " close fail")
		}
	}()
	log.Println("Rpc Serve Listen " + addr)
	rpc.Accept(l)
}
