package cluster

import (
    "log"
)

const BlockSize = 1024

type Client struct {
    Node    *Node
    Cluster *Cluster
}

func NewClient(domain string, name string, group string, partitions int, bind string, port int) (*Client, error) {
    node := NewNode(domain, name)
    node.Bind = bind
    node.Port = port
    node.Group = &group

    node.AnnouncePresence()
    node.StartDiscovery()

    cluster, err := NewVia(node, partitions)

    go func() {
        for {
            select {
            case peer := <- node.Joined:
                if *peer.Name == *node.Name {
                    // stupid but anyways: once I notice I joined, I connect to cluster :D
                    cluster.Connect()
                }
                log.Printf("%s joined with %d partitions", *peer.Name, peer.Partitions)
            case peer := <- node.Left:
                log.Println(*peer.Name, "left")
            }
        }
    }()

    return &Client{
        Node: node,
        Cluster: cluster,
    }, err
}

func (client *Client) Leave() {
    client.Node.AnnounceGroup(nil)
    client.Cluster.Disconnect()
}

func (client *Client) Join(group string) {
    client.Node.AnnounceGroup(&group)
}

func (client *Client) GetGroup() string {
    if client.Node.Group == nil {
        panic("Not a member")
    } else {
        return *client.Node.Group
    }
}

func (client *Client) IsMember() bool {
    return client.Node.Group != nil
}

func (client *Client) SetName(name string) {
    client.Node.AnnounceName(name)
}

func (client *Client) GetName() string {
    return *client.Node.Name;
}

func (client *Client) Ping(node string) int {
    return client.Cluster.Ping(node)
}

func (client *Client) Load(key []byte, consistencyLevel ConsistencyLevel) ([]byte, int) {
    return client.Cluster.Load(key, consistencyLevel)
}

func (client *Client) Store(key []byte, data []byte, consistencyLevel ConsistencyLevel) int {
    if len(data) > BlockSize {
        // load is limited to block size
        return STORE_ERROR
    }
    return client.Cluster.Store(key, data, consistencyLevel)
}

func (client *Client) KeyNodes(key []byte, consistencyLevel ConsistencyLevel) []*Peer {
    return client.Cluster.HashNodes(key, consistencyLevel)
}

func (client *Client) DiscoverPeers() []*Peer {
    if (! client.Node.IsDiscoveryActive() || len(client.Node.Peers) == 0) {
        client.Node.DiscoverPeers()
    }

    return client.Cluster.Peers()
}

func (client *Client) DiscoverGroups() map[string]Data {
    if (! client.Node.IsDiscoveryActive() || len(client.Node.Peers) == 0) {
        client.Node.DiscoverPeers()
    }

    return client.Node.Groups;
}

func (client *Client) Partitions() []*PeerPartition {
    if (! client.Node.IsDiscoveryActive() || len(client.Node.Peers) == 0) {
        client.Node.DiscoverPeers()
    }

    return client.Cluster.Partitions()
}