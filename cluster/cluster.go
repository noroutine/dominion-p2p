// Cluster built on top of node group
package cluster

import (
    "errors"
    "fmt"
    "net"
    "container/list"
)

type Cluster struct {
    proxy *Node
    Server *Server
    Name string
    OrderId uint64
    handlers *list.List
}

// Create a cluster instance with node as a communication proxy
func NewVia(node *Node) (c *Cluster, err error) {
    if ! node.IsOperational() {
        return nil, errors.New("Node is not ready")
    }

    c = &Cluster{
        proxy: node,
        Name: *node.Group,
        handlers: list.New(),
    }
    c.handlers.PushBack(NewPongActivity(c))
    return c, nil
}

// Connects to the cluster and start responding for cluster communications
func (c *Cluster) Connect() {
    // start listening on the DHT
    c.Name = *c.proxy.Group
    serviceEntry := c.proxy.GetServiceEntry()
    c.Server = NewServer(serviceEntry.AddrIPv4, serviceEntry.AddrIPv6, c.proxy.Port, c)
    c.Server.Start()
}

// Disconnect from the cluster and stop responding to cluster communications
func (c *Cluster) Disconnect() {
    if c.Server != nil {
        c.Server.Shutdown()
        c.Server = nil
    }
}

func (c *Cluster) PrimaryNode(o Object) *Peer {
    peers := PeerSorter(c.Peers()).ByHash().Sort()
    objectHash := o.Hash()

    l, r := 0, len(peers)
    if Clockwise(peers[r].Hash(), objectHash, peers[l].Hash()) {
        return peers[r]
    }

    var m int
    for r - l > 1 {
        m = l + (r - l) >> 1

        if Clockwise(peers[m].Hash(), objectHash, peers[r].Hash()) {
            l = m
        } else {
            r = m
        }
    }

    return peers[l]
}

// TODO: put the object into cluster DHT
func (c *Cluster) Put(o Object) {
    // p := c.PrimaryNode(o)
    // find the primary node
    // send message
    panic("put data not implemented")
}

// TODO: get the object into cluster DHT
func (c *Cluster) Get(key string) Object {
    // calculate key
    // find the primary node
    // send message
    // wait for response
    panic("get data not implemented")
}

// Route cluster request to concrete handler, makes Cluster a Router
func (c *Cluster) Route(r *Request) (h Handler, err error) {
    for e := c.handlers.Front(); e != nil; e = e.Next() {
        h, err := e.Value.(Router).Route(r)
        if h != nil && err == nil {
            return h, nil
        }
    }

    return nil, errors.New("Not supported")
}

// Ping another cluster peer
func (c *Cluster) Ping(peer string) int {
    activity := NewPingActivity(c)

    e := c.handlers.PushBack(activity)
    defer c.handlers.Remove(e)

    activity.Run(peer)
    return <- activity.Result
}

// Send cluster message as UDP packet
func (c *Cluster) Send(to *net.UDPAddr, m *Message) error {
    udpCl, err := NewClient(to)
    if err != nil {
        return err
    }

    defer udpCl.Close()

    return udpCl.Send(m)
}

func (c *Cluster) Peers() []*Peer {
    // TODO: potential shared memory access
    peersMap := c.proxy.Peers
    peers := make([]*Peer, 0, len(peersMap))
    for _, p := range peersMap {
        peers = append(peers, &p)
    }

    return PeerSorter(peers).ByHash().Sort()
}

// Resolve cluster peer IP address by peer name
func (c *Cluster) GetPeerAddr(peer string) (*net.UDPAddr, error) {
    p, ok := c.proxy.Peers[peer]
    if !ok {
        return nil, errors.New(fmt.Sprintf("Peer not available: %s", peer))
    }

    return &net.UDPAddr{
        IP: p.GetAddrIPv4(),
        Port: p.Port,
    }, nil
}