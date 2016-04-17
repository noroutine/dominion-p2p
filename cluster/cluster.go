// Cluster built on top of node group
package cluster

import (
    "errors"
    "fmt"
    "net"
    "container/list"
    "os"
)

type Cluster struct {
    proxy *Node
    storage Storage
    Server *Server
    Name string
    OrderId uint64
    handlers *list.List
}

const DefaultPartitions = 127
const partitionsKey string = "partitions"

// Create a cluster instance with node as a communication proxy
func NewVia(node *Node, partitions int) (c *Cluster, err error) {
    if ! node.IsOperational() {
        return nil, errors.New("Node is not ready")
    }

    node.SetText(map[string] string {
        "partitions": fmt.Sprintf("%d", partitions),
    })

    c = &Cluster{
        proxy: node,
        storage: NewInMemoryStorage(),
        Name: *node.Group,
        handlers: list.New(),
    }

    c.handlers.PushBack(NewPongActivity(c))
    c.handlers.PushBack(NewBucketStoreActivity(c))
    c.handlers.PushBack(NewBucketLoadActivity(c))
    return c, nil
}

// Connects to the cluster and start responding for cluster communications
func (c *Cluster) Connect() {
    // start listening on the DHT
    c.Name = *c.proxy.Group

    var addrIPv4, addrIPv6 net.IP
    hostname, err := os.Hostname()
    addrs, err := net.LookupIP(hostname)
    if err != nil {
        // Try appending the host domain suffix and lookup again
        // (required for Linux-based hosts)
        tmpHostName := fmt.Sprintf("%s%s.", hostname, c.proxy.Domain)
        addrs, err = net.LookupIP(tmpHostName)
        if err != nil {
            panic(fmt.Errorf("Could not determine host IP addresses for %s", hostname))
        }
    }

    for i := 0; i < len(addrs); i++ {
        if ipv4 := addrs[i].To4(); ipv4 != nil {
            addrIPv4 = addrs[i]
        } else if ipv6 := addrs[i].To16(); ipv6 != nil {
            addrIPv6 = addrs[i]
        }
    }

    c.Server = NewServer(addrIPv4, addrIPv6, c.proxy.Port, c)
    c.Server.Start()
}

// Disconnect from the cluster and stop responding to cluster communications
func (c *Cluster) Disconnect() {
    if c.Server != nil {
        c.Server.Shutdown()
        c.Server = nil
    }
}

// Returns one primary node and as much consistently determined replication nodes as needed for meeting consistency level
func (c *Cluster) HashNodes(objectHash []byte, level ConsistencyLevel) []*Peer {
    copies := c.Copies(level)
    nodes := make([]*Peer, 0, copies)
    partitions := c.Partitions()

    // first of all determine the primary node
    h, l, r, lenPeers := 0, 0, len(partitions) - 1, len(partitions)
    if Clockwise(partitions[r].Hash(), objectHash, partitions[l].Hash()) {
        h = r
    } else {
        var m int
        for r - l > 1 {
            m = l + (r - l) >> 1

            if Clockwise(partitions[m].Hash(), objectHash, partitions[r].Hash()) {
                l = m
            } else {
                r = m
            }
        }

        h = l
    }

    // build up the array - we take the pivot partition and find as many *other* peers as we need

    for j := 0; len(nodes) < copies; j++ {
        partition := partitions[(h - j + lenPeers) % lenPeers]
        // if partition peer is not in nodes yet - we can add this partition
        peerPresent := false
        for k := 0; k < len(nodes); k++ {
            if partition.Peer == nodes[k] {
                // partition is from already added peer
                peerPresent = true
                break
            }
        }

        if ! peerPresent {
            nodes = append(nodes, partition.Peer)
        }
    }

    return nodes
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

func (c *Cluster) Store(key, data []byte, level ConsistencyLevel) int {
    activity := NewStoreActivity(c, c.AdjustedConsistencyLevel(level))

    e := c.handlers.PushBack(activity)
    defer c.handlers.Remove(e)

    activity.Run(key, data)
    return <- activity.Result
}

func (c *Cluster) Load(key []byte, level ConsistencyLevel) ([]byte, int) {
    adjustedLevel := c.AdjustedConsistencyLevel(level)
    activity := NewLoadActivity(c, adjustedLevel)

    e := c.handlers.PushBack(activity)
    defer c.handlers.Remove(e)

    activity.Run(key)

    data, result := activity.Data, <- activity.Result

    if result == LOAD_PARTIAL_SUCCESS {
        c.Store(key, data, adjustedLevel)
    }

    return data, result
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

func (c *Cluster) Partitions() []*PeerPartition {
    peersMap := c.proxy.Peers
    partitions := make([]*PeerPartition, 0, DefaultPartitions*len(peersMap))

    for _, p := range peersMap {
        pp := p.Clone()
        for i := uint32(0); i < p.Partitions; i++ {
            partitions = append(partitions, &PeerPartition{
                Peer: pp,
                Partition: i,
            })
        }
    }

    return PeerPartitionSorterSorter(partitions).ByHash().Sort()
}

func (c *Cluster) Peers() []*Peer {
    // TODO: potential shared memory access
    peersMap := c.proxy.Peers
    peers := make([]*Peer, 0, len(peersMap))

    for _, p := range peersMap {
        peers = append(peers, p.Clone())
    }

    return peers
}

func (c *Cluster) Quorum() int {
    return len(c.proxy.Peers) + 1
}

func (c *Cluster) Size() int {
    return len(c.proxy.Peers)
}

// Resolve cluster peer IP address by peer name
func (c *Cluster) GetPeerAddr(peer string) (*net.UDPAddr, error) {
    p, ok := c.proxy.Peers[peer]
    if !ok {
        return nil, errors.New(fmt.Sprintf("Peer not available: %s", peer))
    }

    return &net.UDPAddr{
        IP: p.AddrIPv4,
        Port: p.Port,
    }, nil
}