package gotcp

import (
	"bufio"
	l4g "code.google.com/p/log4go"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Error type
var (
	ErrConnClosing   = errors.New("use of closed network connection")
	ErrWriteBlocking = errors.New("write packet was blocking")
	ErrReadBlocking  = errors.New("read packet was blocking")
)

// Conn exposes a set of callbacks for the various events that occur on a connection
type Conn struct {
	srv               *Server
	conn              *net.TCPConn  // the raw connection
	extraData         interface{}   // to save extra data
	closeOnce         sync.Once     // close the conn, once, per instance
	closeFlag         int32         // close flag
	closeChan         chan struct{} // close chanel
	packetSendChan    chan Packet   // packet send chanel
	packetReceiveChan chan Packet   // packeet receive chanel
}

// ConnCallback is an interface of methods that are used as callbacks on a connection
type ConnCallback interface {
	// OnConnect is called when the connection was accepted,
	// If the return value of false is closed
	OnConnect(*Conn) bool

	// OnMessage is called when the connection receives a packet,
	// If the return value of false is closed
	OnMessage(*Conn, Packet) bool

	// OnClose is called when the connection closed
	OnClose(*Conn)
}

// newConn returns a wrapper of raw conn
func newConn(conn *net.TCPConn, srv *Server) *Conn {
	return &Conn{
		srv:               srv,
		conn:              conn,
		closeChan:         make(chan struct{}),
		packetSendChan:    make(chan Packet, srv.config.PacketSendChanLimit),
		packetReceiveChan: make(chan Packet, srv.config.PacketReceiveChanLimit),
	}
}

// GetExtraData gets the extra data from the Conn
func (c *Conn) GetExtraData() interface{} {
	return c.extraData
}

// PutExtraData puts the extra data with the Conn
func (c *Conn) PutExtraData(data interface{}) {
	c.extraData = data
}

// GetRawConn returns the raw net.TCPConn from the Conn
func (c *Conn) GetRawConn() *net.TCPConn {
	return c.conn
}

// Close closes the connection
func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closeFlag, 1)
		close(c.closeChan)
		c.conn.Close()
		c.srv.callback.OnClose(c)
	})
}

// IsClosed indicates whether or not the connection is closed
func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

// AsyncReadPacket async reads a packet, this method will never block
func (c *Conn) AsyncReadPacket(timeout time.Duration) (Packet, error) {
	if c.IsClosed() {
		return nil, ErrConnClosing
	}

	if timeout == 0 {
		select {
		case p := <-c.packetReceiveChan:
			return p, nil

		default:
			return nil, ErrReadBlocking
		}

	} else {
		select {
		case p := <-c.packetReceiveChan:
			return p, nil

		case <-c.closeChan:
			return nil, ErrConnClosing

		case <-time.After(timeout):
			return nil, ErrReadBlocking
		}
	}
}

// AsyncWritePacket async writes a packet, this method will never block
func (c *Conn) AsyncWritePacket(p Packet, timeout time.Duration) error {
	if c.IsClosed() {
		return ErrConnClosing
	}

	if timeout == 0 {
		select {
		case c.packetSendChan <- p:
			return nil

		default:
			return ErrWriteBlocking
		}

	} else {
		select {
		case c.packetSendChan <- p:
			return nil

		case <-c.closeChan:
			return ErrConnClosing

		case <-time.After(timeout):
			return ErrWriteBlocking
		}
	}
}

// Do it
func (c *Conn) Do() {
	if !c.srv.callback.OnConnect(c) {
		return
	}
	c.conn.SetDeadline(time.Now().Add(time.Second * 30))
	go c.handleLoop()
	go c.readStickPackLoop()
	//go c.readLoop()
	go c.writeStickPacketLoop()
}

func (c *Conn) readStickPackLoop() {
	c.srv.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.srv.waitGroup.Done()
	}()

	reader := bufio.NewReader(c.conn)
	unCompleteBuffer := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {

		select {
		case <-c.srv.exitChan:
			return

		case <-c.closeChan:
			return

		default:
		}

		n, err := reader.Read(buffer)
		if err != nil {
			l4g.Info("con read found a error: %v", err)
			return
		}
		if n == 1 && string(buffer[:1]) == "P" {

		}
		if n > 0 {
			//fmt.Println("n is ========================================", n)
			unCompleteBuffer = Unpack(append(unCompleteBuffer, buffer[:n]...), c.packetReceiveChan)

		}

	}
}

func (c *Conn) readLoop() {
	c.srv.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.srv.waitGroup.Done()
	}()

	for {
		select {
		case <-c.srv.exitChan:
			return

		case <-c.closeChan:
			return

		default:
		}

		p, err := c.srv.protocol.ReadPacket(c.conn)
		if err != nil {
			return
		}

		c.packetReceiveChan <- p
	}
}

func (c *Conn) writeLoop() {
	c.srv.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.srv.waitGroup.Done()
	}()

	for {
		select {
		case <-c.srv.exitChan:
			return

		case <-c.closeChan:
			return

		case p := <-c.packetSendChan:
			if _, err := c.conn.Write(DoPacket(p.Serialize())); err != nil {
				return
			}
		}
	}
}

func (c *Conn) writeStickPacketLoop() {
	c.srv.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.srv.waitGroup.Done()
	}()

	for {
		select {
		case <-c.srv.exitChan:
			return

		case <-c.closeChan:
			return

		case p := <-c.packetSendChan:
			if _, err := c.conn.Write(DoPacket(p.Serialize())); err != nil {
				l4g.Info("con write found a error: %v", err)
				return
			}
		}
	}
}

func (c *Conn) handleLoop() {
	c.srv.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.srv.waitGroup.Done()
	}()

	for {
		select {
		case <-c.srv.exitChan:
			return

		case <-c.closeChan:
			return

		case p := <-c.packetReceiveChan:
			log.Println(p.Serialize())
			if !c.srv.callback.OnMessage(c, p) {
				return
			}
		}
	}
}
