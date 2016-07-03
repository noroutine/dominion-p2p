// Simple wrapper of UDP client that allows to send cluster messages over the network
package cluster

import (
    "log"
    "net"
)

type UdpClient struct {
    ipv4conn *net.UDPConn
}

func NewUdpClient(raddr *net.UDPAddr) (*UdpClient, error) {
    c, err := net.DialUDP("udp4", nil, raddr)

    if err != nil {
        log.Println("Cannot connect to", raddr)
        return nil, err
    }

    return &UdpClient{
        ipv4conn: c,
    }, nil
}

func (c *UdpClient) Send(m *Message) error {
    raw := Marshall(m)
    w, err := c.ipv4conn.Write(raw)
    if w < len(raw) || err != nil {
        log.Println("error writing message", w, len(raw), err)
    }
    return err
}

func (c *UdpClient) Close() {
    c.ipv4conn.Close()
}

