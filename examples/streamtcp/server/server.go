package main

import (
	//"fmt"
	l4g "code.google.com/p/log4go"
	"github.com/whiskerman/gotcp"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Callback struct {
	cm *gotcp.ConnManager
}

func (this *Callback) OnConnect(c *gotcp.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	c.PutExtraData(addr.String())
	//str := addr.String()
	/* act on str */
	this.cm.AddConn(addr.String(), c)

	log.Println("OnConnect:", addr)
	return true
}

func (this *Callback) OnMessage(c *gotcp.Conn, p gotcp.Packet) bool {
	streamPacket := p.(*gotcp.StreamPacket)
	log.Printf("from %p OnMessage:[%v] [%v]\n", c, streamPacket.GetLength(), string(streamPacket.GetBody()))
	this.cm.Broadcast(gotcp.NewStreamPacket(streamPacket.Serialize()))
	//c.AsyncWritePacket(gotcp.NewStreamPacket(streamPacket.Serialize()), time.Second)
	return true
}

func (this *Callback) OnClose(c *gotcp.Conn) {
	log.Println("OnClose:", c.GetExtraData())

	addr := c.GetExtraData()
	this.cm.DelConn(addr)

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

	go func() {
		log.Println(http.ListenAndServe("192.168.1.107:6061", nil))

	}()

	defer l4g.Close()
	// creates a server
	config := &gotcp.Config{
		PacketSendChanLimit:    20,
		PacketReceiveChanLimit: 20,
	}
	srv := gotcp.NewServer(config, &Callback{cm: gotcp.NewConnManager()}, nil)

	// starts service
	go srv.Start(listener, time.Second)
	l4g.Debug("listening: %s", listener.Addr())
	//log.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2) //, syscall.SIGINT, syscall.SIGTERM
	sig := <-chSig
	l4g.Trace("Signal: %s", sig)
	//log.Println("Signal: ", <-chSig)

	// stops service
	srv.Stop()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
