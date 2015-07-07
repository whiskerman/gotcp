package gotcp

import (
	l4g "code.google.com/p/log4go"

	"sync"
)

type ConnManager struct {
	rwlock  sync.RWMutex
	connMap map[string]*Conn
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		connMap: make(map[string]*Conn, 40000),
	}
}

func (m *ConnManager) AddConn(id string, conn *Conn) {
	m.rwlock.Lock()
	defer m.rwlock.Unlock()
	m.connMap[id] = conn
	l4g.Debug("set %s conn %p", id, conn)
}

func (m *ConnManager) DelConn(id string) {
	m.rwlock.Lock()
	defer m.rwlock.Unlock()
	conn := m.connMap[id]
	delete(m.connMap, id)
	l4g.Debug("delete %s conn %p", id, conn)
}

func (m *ConnManager) Broadcast(packet Packet) {
	for _, conn := range m.connMap {
		if !conn.IsClosed() {
			conn.AsyncWritePacket(packet, 0)
			l4g.Info("send to %p, packet is %v", conn, packet)
		}

	}
}
