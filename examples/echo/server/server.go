package main

import (
	//"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	l4g "code.google.com/p/log4go"
	"github.com/gansidui/gotcp"
	"github.com/gansidui/gotcp/examples/echo"
)

type Callback struct{}

func (this *Callback) OnConnect(c *gotcp.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	c.PutExtraData(addr)
	log.Println("OnConnect:", addr)
	return true
}

func (this *Callback) OnMessage(c *gotcp.Conn, p gotcp.Packet) bool {
	echoPacket := p.(*echo.EchoPacket)
	log.Printf("OnMessage:[%v] [%v]\n", echoPacket.GetLength(), string(echoPacket.GetBody()))
	c.AsyncWritePacket(echo.NewEchoPacket(echoPacket.Serialize(), true), time.Second)
	return true
}

func (this *Callback) OnClose(c *gotcp.Conn) {
	log.Println("OnClose:", c.GetExtraData())
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	//l4g.AddFilter("file", l4g.FINE, l4g.NewFileLogWriter("server.log", false))
	// creates a tcp listener
	l4g.LoadConfiguration("./conf/log4go.xml")
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8989")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)
	defer l4g.Close()
	// creates a server
	config := &gotcp.Config{
		PacketSendChanLimit:    20,
		PacketReceiveChanLimit: 20,
	}
	srv := gotcp.NewServer(config, &Callback{}, &echo.EchoProtocol{})

	// starts service
	go srv.Start(listener, time.Second)
	l4g.Debug("listening: %s", listener.Addr())
	//log.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2) //, syscall.SIGINT, syscall.SIGTERM
	l4g.Trace("Signal: %s", <-chSig)
	//log.Println("Signal: ", <-chSig)

	// stops service
	srv.Stop()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
