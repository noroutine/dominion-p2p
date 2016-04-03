package cluster

import (
    "testing"
)

func TestMarshalling(t *testing.T) {
    m := Message{
        Version:    1,
        Type:       PING,
        Operation:  0,
        Args:       make([]byte, 8, 8),
        Length:     1,
        Load:       []byte { 1 },
    }

    m1, err := Unmarshall(Marshall(&m))

    if len(m.ReplyTo) > 0 {
        t.Error("No ReplyTo was marshalled")
    }

    if err != nil {
        t.Fail()
    }

    if m1.Version != m.Version {
        t.Fail()
    }

    if m1.Operation != m.Operation {
        t.Fail()
    }

    if m1.Type != m.Type {
        t.Fail()
    }

    if m1.Length != m.Length {
        t.Fail()
    }

    if m1.Load[0] != m.Load[0] {
        t.Fail()
    }

    if m1.ReplyTo != m.ReplyTo {
        t.Fail()
    }
}

func TestReplyTo(t *testing.T) {
    replyTo := "me"

    m := &Message{
        Version:    1,
        Type:       PING,
        Operation:  0,
        Args:       make([]byte, 8, 8),
        ReplyTo:    replyTo,
        Length:     0,
        Load:       []byte { 0 },
    }

    raw := Marshall(m)

    m1, err := Unmarshall(raw)

    if err != nil {
        t.Error(err)
    }

    if m1.ReplyTo != m.ReplyTo {
        t.Fail()
    }

    // byte 36 + 2 + 1 shall be zero
    if raw[36 + len(replyTo) + 1] > 0 {
        t.Error("Invalid zero-termination of ReplyTo")
    }

    // byte 291 shall be zero in any case
    if raw[291] > 0 {
        t.Error("Marshalling error - byte 291 shall be zero by design")
    }

}
