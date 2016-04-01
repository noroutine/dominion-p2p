// Peer is what another node is to current node
package cluster

import (
    "net"
    "github.com/noroutine/bonjour"
    "github.com/reusee/mmh3"
    "math/big"
    "sort"
)

const hash_byte_len = 16        // 128 / 8

type Peer struct {
    Domain *string
    Name *string
    HostName *string
    Port int
    Group *string

    entry *bonjour.ServiceEntry
}

func (p *Peer) GetAddrIPv4() net.IP {
    return p.entry.AddrIPv4
}

func (p *Peer) GetAddrIPv6() net.IP {
    return p.entry.AddrIPv6
}

func (p *Peer) Hash() []byte {
    return mmh3.Sum128([]byte(*p.Name))
}

type peerSorter struct {
    peers []*Peer
    less func (*Peer, *Peer) bool
}

func PeerSorter(ps []*Peer) *peerSorter {
    return &peerSorter{
        peers: ps,
        less: nil,
    }
}

func (sorter *peerSorter) ByHash() *peerSorter {
    sorter.less = func (p1, p2 *Peer) bool {
        iHash := new(big.Int).SetBytes(p1.Hash())
        jHash := new(big.Int).SetBytes(p2.Hash())
        return iHash.Cmp(jHash) < 0
    }

    return sorter
}

func (sorter *peerSorter) Sort() []*Peer {
    sort.Sort(sorter)
    return sorter.peers
}

func (hs *peerSorter) Len() int {
    return len(hs.peers)
}

func (hs *peerSorter) Swap(i, j int) {
    ps := hs.peers
    ps[i], ps[j] = ps[j], ps[i]
}

func (sorter *peerSorter) Less(i, j int) bool {
    return sorter.less(sorter.peers[i], sorter.peers[j])
}