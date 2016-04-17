// Peer is what another node is to current node
package cluster

import (
    "net"
    "github.com/reusee/mmh3"
    "math/big"
    "sort"
    "strings"
)

const hash_byte_len = 16        // 128 / 8

type Peer struct {
    Domain *string
    Name *string
    Partitions uint32
    HostName *string
    Port int
    Group *string
    AddrIPv4 net.IP
    AddrIPv6 net.IP
    Text []string
}

type PeerPartition struct {
    Peer *Peer
    Partition uint32
}

func (p *Peer) Clone() *Peer {
    return &Peer{
        Domain: p.Domain,
        Name: p.Name,
        Partitions: p.Partitions,
        HostName: p.HostName,
        Port: p.Port,
        Group: p.Group,
        AddrIPv4: p.AddrIPv4,
        AddrIPv6: p.AddrIPv6,
        Text: p.Text,
    }
}

func (p *Peer) getText(key string) *string {
    ketEq := key + "="
    for _, s := range p.Text {
        if strings.HasPrefix(s, ketEq) {
            value := strings.TrimPrefix(s, ketEq)
            return &value
        }
    }

    return nil
}

func (p *PeerPartition) Hash() []byte {
    return mmh3.Sum128(append([]byte(*p.Peer.Name),
        byte(p.Partition >> 24),
        byte(p.Partition >> 16),
        byte(p.Partition >> 8),
        byte(p.Partition),
    ))
}

type peerPartitionSorter struct {
    partitions []*PeerPartition
    less       func (*PeerPartition, *PeerPartition) bool
}

func PeerPartitionSorterSorter(ps []*PeerPartition) *peerPartitionSorter {
    return &peerPartitionSorter{
        partitions: ps,
        less: nil,
    }
}

func (sorter *peerPartitionSorter) ByHash() *peerPartitionSorter {
    sorter.less = func (p1, p2 *PeerPartition) bool {
        iHash := new(big.Int).SetBytes(p1.Hash())
        jHash := new(big.Int).SetBytes(p2.Hash())
        return iHash.Cmp(jHash) < 0
    }

    return sorter
}

func (sorter *peerPartitionSorter) Sort() []*PeerPartition {
    sort.Sort(sorter)
    return sorter.partitions
}

func (hs *peerPartitionSorter) Len() int {
    return len(hs.partitions)
}

func (hs *peerPartitionSorter) Swap(i, j int) {
    ps := hs.partitions
    ps[i], ps[j] = ps[j], ps[i]
}

func (sorter *peerPartitionSorter) Less(i, j int) bool {
    return sorter.less(sorter.partitions[i], sorter.partitions[j])
}