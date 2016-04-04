// Cluster message structure and marshalling
package cluster

import (
    "errors"
)

type OperationType byte

const (
    NOOP  OperationType = iota
    PING                          // ping operations
    JOIN                          // join operations
    STORE                         // store operations
    LOAD                          // load operations
)

type Message struct {
    Version     byte               //   1 byte
    Type        OperationType      // + 1 bytes    = 2
    Operation   byte               // + 1 bytes    = 3
//  reserved1   byte               // + 1 byte     = 4
    Args        []byte             // + 16 bytes   = 20
//  reserved2   []byte             // + 16 bytes   = 36
    ReplyTo     string             // + 256 bytes  = 292
    Length      uint16             // + 2 bytes    = 294
    Load        []byte
}

const HeaderSize = 294
const MaxLoadLength = 0xFFFF - HeaderSize

func Unmarshall(packet []byte) (m *Message, err error) {
    if len(packet) < HeaderSize {
        err = errors.New("Packet is too small")
        return
    }

    replyTo := make([]byte, 0, 255)
    // last byte shall be only \0 always, so we ignore it
    for _, b := range packet[36:291] {
        if b == 0 {
            break
        }

        replyTo = append(replyTo, b)
    }

    return &Message{
        Version:    packet[0],
        Type:       OperationType(packet[1]),
        Operation:  packet[2],
        Args:       packet[4:20],
        ReplyTo:    string(replyTo),
        Length:     uint16(packet[292]) << 8 | uint16(packet[293]),
        Load:       packet[294:],
    }, nil
}

func Marshall(m *Message) []byte {
    l := HeaderSize + len(m.Load)
    buf := make([]byte, l, l)
    buf[0] = m.Version
    buf[1] = byte(m.Type)

    buf[2] = byte(m.Operation)
    // byte 3 is reserved

    if len(m.Args) > 16 {
        panic("Too many message arguments, max 16 allowed")
    }
    for i := range(m.Args) {
        buf[4 + i] = m.Args[i]
    }
    // the rest is filled with zeroes
    for i := len(m.Args); i < 16; i++ {
        buf[4 + i] = 0
    }

    // bytes 20 to 35 are reserved

    if len(m.ReplyTo) > 255 {
        panic("ReplyTo shall be max 255 bytes")
    }

    for i, b := range []byte(m.ReplyTo) {
        buf[36 + i] = b
    }
    // the rest is filled with zeroes
    for i := len(m.ReplyTo); i < 255; i++ {
        buf[36 + i] = 0
    }

    buf[292] = byte(m.Length >> 8)
    buf[293] = byte(m.Length)

    if len(m.Load) > MaxLoadLength {
        panic("Message data is too big")
    }

    for i, p := range m.Load {
        buf[294 + i] = p
    }

    return buf
}