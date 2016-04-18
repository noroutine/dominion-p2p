package cluster

import (
    "net"
    "log"
)

type Request struct {
    From *net.UDPAddr
    Message *Message
}

type Handler interface {
    Handle(*Request) error
}

type Router interface {
    Route(*Request) (Handler, error)
}

type Server struct {
    ipv4conn *net.UDPConn
    ipv6conn *net.UDPConn
    shouldShutdown bool
    router Router
}

func NewServer(port int, r Router) *Server {
    l4, err := net.ListenUDP("udp4", &net.UDPAddr{ Port: port })
    if err != nil {
        log.Println(err)
    }

    return &Server{
        ipv4conn: l4,
        ipv6conn: nil,
        shouldShutdown: false,
        router: r,
    }
}

func (s *Server) Start() {
    go s.serve(s.ipv4conn)    
    go s.serve(s.ipv6conn)    
}

func (s *Server) Shutdown() {
    s.shouldShutdown = true

    if s.ipv6conn != nil {
        s.ipv6conn.Close()
    }

    if s.ipv4conn != nil {
        s.ipv4conn.Close()
    }
}

func (s *Server) serve(c *net.UDPConn) {
    if c == nil {
        return
    }

    buf := make([]byte, 65536)
    
    for {
        n, from, err := c.ReadFromUDP(buf)
        if err != nil {
             log.Println(err)
             continue
        }

        m, err := Unmarshall(buf[:n])
        if err != nil {
            log.Println(err)
            continue
        }

        r := &Request{
            From: from,
            Message: m,
        }

        h, err := s.router.Route(r)
        if err != nil {
            log.Println(err)
            continue
        }

        err = h.Handle(r)
        if err != nil {
            log.Println(err)
        }
    }
}
