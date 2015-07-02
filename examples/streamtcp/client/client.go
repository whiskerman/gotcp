package main

import (
	"bufio"
	//"fmt"
	"github.com/whiskerman/gotcp"
	"log"
	"net"
	"time"
)

const (
	clientscount = 30000
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8989")
	checkError(err)
	conns := createclients(clientscount, tcpAddr)
	//echoProtocol := nil

	// ping <--> pong
	for i := 0; i < clientscount; i++ {
		// write
		outstr := gotcp.DoPacket([]byte("hello"))
		log.Printf("out str is %s", outstr)
		conns[i].Write(outstr)

		// read

		//time.Sleep(2 * time.Second)
	}

	time.Sleep(1200 * time.Second)
	for i := 0; i < clientscount; i++ {
		conns[i].Close()
	}

	log.Println("ending client.go")

}

func createclients(x int, addr *net.TCPAddr) []*net.TCPConn {
	result := make([]*net.TCPConn, x)
	for m := 0; m < x; m++ {
		conn, err := net.DialTCP("tcp", nil, addr)
		checkError(err)
		readchan := make(chan gotcp.Packet)
		go readStickPackLoop(conn, readchan)
		result[m] = conn
		go func(ch chan gotcp.Packet) {
			p, ok := <-ch //echoProtocol.ReadPacket(conn)
			if ok {
				streamPacket := p.(*gotcp.StreamPacket)
				log.Printf("Server reply:[%v] [%v]\n", streamPacket.GetLength(), string(streamPacket.GetBody()))
			}
		}(readchan)

	}
	return result
}

func readStickPackLoop(c *net.TCPConn, rchan chan gotcp.Packet) {

	defer func() {
		recover()
		c.Close()

	}()

	reader := bufio.NewReader(c)
	unCompleteBuffer := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		/*
			select {
			case <-c.srv.exitChan:
				return

			case <-c.closeChan:
				return

			default:
			}
		*/
		n, err := reader.Read(buffer)
		if err != nil {

			return
		}
		if n == 1 && string(buffer[:1]) == "P" {

		}
		if n > 0 {
			//fmt.Println("n is ========================================", n)
			unCompleteBuffer = gotcp.Unpack(append(unCompleteBuffer, buffer[:n]...), rchan)

		}

	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
