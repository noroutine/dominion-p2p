package cluster

import (
    "errors"
    "log"
    "fmt"
    "encoding/gob"
    "github.com/noroutine/dominion/fsa"
    "github.com/reusee/mmh3"
    "bytes"
)

const (
    STORE_START int = iota
    STORE_SEND
    STORE_WAIT_ACK
    STORE_RCVD_ACK
    STORE_FULL_ACK
    STORE_PARTIAL_ACK
    STORE_NO_ACK
    STORE_SUCCESS
    STORE_PARTIAL_SUCCESS
    STORE_FAILURE
    STORE_ERROR
)

type DataTransferObject struct {
    Key []byte
    Value []byte
}

type BucketActivity struct {
    c *Cluster
}

func NewBucketActivity(c *Cluster) *BucketActivity {
    return &BucketActivity{
        c: c,
    }
}

func (a *BucketActivity) Route(r *Request) (h Handler, err error) {
    // reply to ping with pong
    if r.Message.Type == STORE && r.Message.Operation == 0 {
        return a, nil
    }

    return nil, errors.New("Cannot handle this")
}

func (a *BucketActivity) Handle(r *Request) error {

    peer := string(r.Message.ReplyTo)
    ackAddr, err := a.c.GetPeerAddr(peer)
    if err != nil {
        return errors.New(fmt.Sprintf("Cannot ack request from %s", peer))
    }

    raw := bytes.NewBuffer(r.Message.Load)
    dec := gob.NewDecoder(raw)
    var dto DataTransferObject
    err = dec.Decode(&dto)
    if err != nil {
        log.Fatal("Decode error: ", err)
    }

    a.c.storage.Put(dto.Key, dto.Value)

    log.Printf("Got %d bytes of data to store, sending ack to %s", r.Message.Length, peer)

    // send pong back
    go a.c.Send(ackAddr, &Message{
        Version: 1,
        Type: STORE,
        Operation: 1,   // ack
        Length: uint16(len(*a.c.proxy.Name)),
        Load: []byte(*a.c.proxy.Name),
    })

    return nil
}

type StoreActivity struct {
    c *Cluster
    fsa *fsa.FSA
    acks int
    Result chan int
}

func NewStoreActivity(c *Cluster) *StoreActivity {
    return &StoreActivity{
        Result: make(chan int, 1),
        acks: 2,
        c: c,
        fsa: nil,
    }
}

func (a *StoreActivity) Route(r *Request) (h Handler, err error)  {
    if r.Message.Type == STORE && r.Message.Operation == 1 {
        return a, nil
    }

    return nil, errors.New("Cannot handle this")
}

func (a *StoreActivity) Handle(r *Request) error {
    log.Println("Received ack from ", string(r.Message.Load))
    go a.fsa.Send(STORE_RCVD_ACK)
    return nil
}

func (a *StoreActivity) Run(key, data []byte) {
    a.fsa = fsa.New(func(state, input int) int {
        switch{
        case state == STORE_START && input == STORE_START:
            go a.fsa.Send(STORE_SEND)
            return STORE_SEND
        case state == STORE_SEND && input == STORE_SEND:

            raw := new(bytes.Buffer)
            enc := gob.NewEncoder(raw)
            err := enc.Encode(DataTransferObject {
                Key: key,
                Value: data,
            })

            if err != nil {
                panic(fmt.Sprintf("Can't encode data for transfer: %v", err))
            }

            if raw.Len() > MaxLoadLength {
                panic("Load is too big")
            }

            primary, secondary := a.c.HashNodes(mmh3.Sum128(key))
            primaryAddr, err := a.c.GetPeerAddr(*primary.Name)
            if err != nil {
                log.Println("Cannot contact peer", *primary.Name)
                a.Result <- STORE_ERROR
                return STORE_ERROR
            }

            secondaryAddr, err := a.c.GetPeerAddr(*secondary.Name)
            if err != nil {
                log.Println("Cannot contact peer", *secondary.Name)
                a.Result <- STORE_ERROR
                return STORE_ERROR
            }

            // send store command to primary and secondary nodes
            m := &Message{
                Version: 1,
                Type: STORE,
                Operation: 0,   // store
                ReplyTo: *a.c.proxy.Name,
                Length: uint16(raw.Len()),
                Load: raw.Bytes(),
            }

            go a.c.Send(primaryAddr, m)
            go a.c.Send(secondaryAddr, m)

            return STORE_WAIT_ACK
        case state == STORE_WAIT_ACK && input == STORE_RCVD_ACK:
            a.acks--
            if a.acks == 0 {
                go a.fsa.Send(STORE_FULL_ACK)
                return STORE_FULL_ACK
            } else {
                return STORE_WAIT_ACK
            }
        case state == STORE_NO_ACK:
            a.Result <- STORE_FAILURE
            return STORE_FAILURE
        case state == STORE_PARTIAL_ACK:
            a.Result <- STORE_PARTIAL_SUCCESS
            return STORE_PARTIAL_SUCCESS
        case state == STORE_FULL_ACK:
            a.Result <- STORE_SUCCESS
            return STORE_SUCCESS
        }
        log.Println("Invalid automat")
        a.Result <- STORE_ERROR
        return STORE_ERROR
    }, fsa.TerminatesOn(STORE_SUCCESS, STORE_PARTIAL_SUCCESS, STORE_FAILURE), fsa.NeverTimesOut())

    go a.fsa.Send(PING_START)
}