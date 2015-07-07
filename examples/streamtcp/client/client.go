package main

import (
	"bufio"
	l4g "code.google.com/p/log4go"
	//"fmt"
	"github.com/whiskerman/gotcp"
	"io"
	"log"
	"net"
	"time"
)

const (
	clientscount = 40000
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "192.168.122.132:8989")
	checkError(err)
	conns := createclients(clientscount, tcpAddr)
	//echoProtocol := nil

	// ping <--> pong
	for i := 0; i < clientscount; i++ {
		// write
		outstr := gotcp.DoPacket([]byte("hello"))
		log.Printf("out str is %s", outstr)
		conns[i].SetWriteDeadline(time.Now().Add(time.Second * 30))
		conns[i].Write(outstr)

		// read

		//time.Sleep(2 * time.Second)
	}

	time.Sleep(1200 * time.Second)
	for i := 0; i < clientscount; i++ {
		conns[i].Close()
		log.Println("conn close :", i)
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
		go func(ch chan gotcp.Packet, con *net.TCPConn) {
			p, ok := <-ch //echoProtocol.ReadPacket(conn)
			if ok {
				streamPacket := p.(*gotcp.StreamPacket)
				log.Printf("connection: %p Server reply:[%v] [%v]\n", conn, streamPacket.GetLength(), string(streamPacket.GetBody()))
			}
		}(readchan, conn)

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
		c.SetReadDeadline(time.Now().Add(time.Second * 30))
		n, err := reader.Read(buffer)

		if e, ok := err.(net.Error); ok && e.Timeout() {
			continue
			// This was a timeout
		} else if err != nil {
			if err == io.EOF {
				l4g.Info("client read a eof error: %v", err)
			}

			l4g.Info("client read a error: %v", err)
			return
			// This was an error, but not a timeout

		}

		if n == 1 && string(buffer[:1]) == "P" {
			l4g.Debug("connection %p reciev ping", c)

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
