package protocol

import (
	"net"
	"sync"

	"github.com/stampzilla/stampzilla-go/protocol"
)

//global variable to hold our nodes. Initialized in main
var nodes *Nodes

type Node struct {
	protocol.Node
	conn net.Conn
	wait chan bool
	sync.RWMutex
}

func (n *Node) Conn() net.Conn {
	n.Lock()
	defer n.Unlock()
	return n.conn
}
func (n *Node) SetConn(conn net.Conn) {
	n.Lock()
	n.conn = conn
	n.Unlock()
}

//  TODO: write tests for Nodes struct (jonaz) <Fri 10 Oct 2014 04:31:22 PM CEST>
type Nodes struct {
	nodes map[string]*Node
	sync.RWMutex
}

func NewNodes() *Nodes {
	n := &Nodes{}
	n.nodes = make(map[string]*Node)
	return n
}

func (n *Nodes) ByName(name string) *Node {
	n.RLock()
	defer n.RUnlock()
	for _, node := range n.nodes {
		if node.Name == name {
			return node
		}
	}
	return nil
}
func (n *Nodes) Search(nameoruuid string) *Node {
	if n := n.ByUuid(nameoruuid); n != nil {
		return n
	}
	if n := n.ByName(nameoruuid); n != nil {
		return n
	}
	return nil
}
func (n *Nodes) ByUuid(uuid string) *Node {
	n.RLock()
	defer n.RUnlock()
	if node, ok := n.nodes[uuid]; ok {
		return node
	}
	return nil
}
func (n *Nodes) All() map[string]*Node {
	n.RLock()
	defer n.RUnlock()
	return n.nodes
}
func (n *Nodes) Add(node *Node) {
	n.Lock()
	defer n.Unlock()
	//var newNode = node
	n.nodes[node.Uuid] = node
}
func (n *Nodes) Delete(uuid string) {
	n.Lock()
	defer n.Unlock()
	delete(n.nodes, uuid)
}