package main

import (
	"bufio"
	//"fmt"
	"github.com/whiskerman/gotcp"
	"log"
	"net"
	"time"
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8989")
	checkError(err)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	readchan := make(chan gotcp.Packet)
	//echoProtocol := nil
	go readStickPackLoop(conn, readchan)

	// ping <--> pong
	for i := 0; i < 40000; i++ {
		// write
		outstr := gotcp.DoPacket([]byte("hello"))
		log.Printf("out str is %s", outstr)
		conn.Write(outstr)

		// read

		//time.Sleep(2 * time.Second)
	}

	go func() {
		p, ok := <-readchan //echoProtocol.ReadPacket(conn)
		if ok {
			streamPacket := p.(*gotcp.StreamPacket)
			log.Printf("Server reply:[%v] [%v]\n", streamPacket.GetLength(), string(streamPacket.GetBody()))
		}
	}()
	time.Sleep(120 * time.Second)
	conn.Close()
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
