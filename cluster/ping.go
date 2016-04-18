// Implementation of ping operation
package cluster

import (
    "log"
    "fmt"
    "time"
    "errors"
    "github.com/noroutine/dominion/fsa"
)

const (
    PING_START = iota
    PING_SENT
    PING_WAIT_PONG
    PING_RCVD_PONG
    PING_SUCCESS
    PING_TIMEOUT
    PING_ERROR
)

type PongActivity struct {
    c *Cluster
}

func NewPongActivity(c *Cluster) *PongActivity {
    return &PongActivity{
        c: c,
    }
}

func (a *PongActivity) Route(r *Request) (h Handler, err error) {
    // reply to ping with pong
    if r.Message.Type == PING && r.Message.Operation == 0 {
        return a, nil
    }

    return nil, errors.New("Cannot handle this")
}

func (a *PongActivity) Handle(r *Request) error {
    peer := string(r.Message.Load)
    pongAddr, err := a.c.GetPeerAddr(peer)
    if err != nil {
       return errors.New(fmt.Sprintf("Cannot pong peer %s", peer))
    }

    // send pong back
    go a.c.Send(pongAddr, &Message{
        Version: 1,
        Type: PING,
        Operation: 1,   // pong
        Length: 0,
        Load: make([]byte, 0, 0),
    })

    return nil
}

type PingActivity struct {
    Result chan int
    c *Cluster
    fsa *fsa.FSA
}

func NewPingActivity(c *Cluster) *PingActivity {
    return &PingActivity{
        Result: make(chan int, 1),
        c: c,
        fsa: nil,
    }
}

func (a *PingActivity) Route(r *Request) (h Handler, err error)  {
    if r.Message.Type == PING && r.Message.Operation == 1 {
        return a, nil
    }

    return nil, errors.New("Cannot handle this")
}

func (a *PingActivity) Handle(r *Request) error {
    go a.fsa.Send(PING_RCVD_PONG)
    return nil
}

func (a *PingActivity) Run(target string) {
    timeoutFunc := func(state int) (<-chan time.Time, func(int) int) {
        if state == PING_WAIT_PONG {
            return time.After(1000*time.Millisecond), func(s int) int {
                a.Result <- PING_TIMEOUT
                return PING_TIMEOUT
            }
        }

        return fsa.NeverTimesOut()(state)
    }

    a.fsa = fsa.New(func(state, input int) int {
        switch{
        case state == PING_START && input == PING_START:
            targetAddr, err := a.c.GetPeerAddr(target)
            if err != nil {
                log.Println("Cannot ping peer")
                a.Result <- PING_ERROR
                return PING_ERROR
            }

            // send ping 
            go a.c.Send(targetAddr, &Message{
                Version: 1,
                Type: PING,
                Operation: 0,   // ping
                Length: uint16(len(*a.c.proxy.Name)),
                Load: []byte(*a.c.proxy.Name),
            })

            go a.fsa.Send(PING_SENT)
            return PING_SENT
        case state == PING_SENT && input == PING_SENT:
            return PING_WAIT_PONG
        case state == PING_WAIT_PONG && input == PING_RCVD_PONG:
            go a.fsa.Send(PING_SUCCESS)
            return PING_RCVD_PONG
        case state == PING_RCVD_PONG && input == PING_SUCCESS:
            a.Result <- PING_SUCCESS
            return PING_SUCCESS
        }
        log.Println("Invalid automat")
        a.Result <- PING_ERROR
        return PING_ERROR
    }, fsa.TerminatesOn(PING_TIMEOUT, PING_SUCCESS, PING_ERROR), timeoutFunc)

    go a.fsa.Send(PING_START)
}