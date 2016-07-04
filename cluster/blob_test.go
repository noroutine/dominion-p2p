package cluster

import (
    "testing"
    "log"
    "time"
    "flag"
    "os"
)

var client1, client2, client3 *Client

func TestMain(m *testing.M) {
    client1, client2, client3, _, _ = startTestCluster()
    flag.Parse()
    os.Exit(m.Run())
}

func TestGetPageKey(t *testing.T) {
    blobKey := []byte("testKey")

    blob := &Blob{
        client: nil,
        key: blobKey,
        pos: 0,
        pageSize: BlockSize,
        pages: make(map[int][]byte),
    }

    pageKey := string(blob.GetPageKey(0))

    if pageKey != "testKey.00000000" {
        t.Fatal(pageKey)
    }

    pageKey = string(blob.GetPageKey(1))

    if pageKey != "testKey.00000000" {
        t.Fatal(pageKey)
    }

    pageKey = string(blob.GetPageKey(BlockSize))

    if pageKey != "testKey.00000001" {
        t.Fatal(pageKey)
    }

    pageKey = string(blob.GetPageKey(2 * BlockSize))

    if pageKey != "testKey.00000002" {
        t.Fatal(pageKey)
    }

}

func TestBlob_OpenBlob(t *testing.T) {
    _, err := client1.OpenBlob([]byte("test"), ConsistencyLevelTwo)
    if err == nil {
        t.Fatal("Opening non-existing blob shall fail")
    }
}

func TestBlob_CreateBlob(t *testing.T) {

    var testBlobSize int64 = 100

    _, err := client1.CreateBlob([]byte("test123"), testBlobSize, ConsistencyLevelTwo)
    if err != nil {
        t.Fatal("Creating non-existing blob shall succeed", err)
    }

    blob, err := client1.OpenBlob([]byte("test123"), ConsistencyLevelTwo)

    if err != nil {
        t.Fatal("Opening existing blob shall succeed", err)
    }

    if blob.Size() != testBlobSize {
        t.Fatal("Blob size mismatch", testBlobSize, blob.Size())
    }

}

func TestBlob_WriteByte(t *testing.T) {

    var testByte byte = 1
    var testPosition int64 = 2
    var testBlobSize int64 = 1 + testPosition

    blob1, err := client1.CreateBlob([]byte("test"), testBlobSize, ConsistencyLevelTwo)
    if err != nil {
        t.Fatal("Failed to open blob", err)
    }

    blob2, _ := client2.OpenBlob([]byte("test"), ConsistencyLevelTwo)
    client3.OpenBlob([]byte("test"), ConsistencyLevelTwo)

    if blob2.Size() != testBlobSize {
        t.Fatal("Blob size mismatch", blob2.Size(), testBlobSize)
    }

    log.Println("Writing 1 byte")
    if err := blob1.WriteByteAt(testByte, testPosition); err != nil {
        t.Fatal("Error writing byte", err)
    }
    log.Println("Reading 1 byte")
    if c, err := blob2.ReadByteAt(testPosition); err != nil {
        t.Fatal("Error reading byte", err)
    } else {
        if c != testByte {
            t.Fatal("Byte read does not match byte written", c, testByte)
        } else {
            log.Println("Byte write OK!")
        }
    }
}

func TestBlob_WriteSmall(t *testing.T) {
    var testBytes []byte = []byte("Hello World!")
    var testPosition int64 = 2
    var testBlobSize int64 = int64(len(testBytes)) + testPosition

    blob1, err := client1.CreateBlob([]byte("test"), testBlobSize, ConsistencyLevelTwo)
    if err != nil {
        t.Fatal("Failed to open blob")
    }

    blob2, _ := client2.OpenBlob([]byte("test"), ConsistencyLevelTwo)
    client3.OpenBlob([]byte("test"), ConsistencyLevelTwo)

    log.Println("Writing bytes")
    if n, err := blob1.WriteAt(testBytes, testPosition); err != nil {
        t.Fatal("Error writing bytes, wrote:", n, "error", err)
    }
    log.Println("Reading bytes back")
    p := make([]byte, len(testBytes))
    if n, err := blob2.ReadAt(p, testPosition); err != nil {
        t.Fatal("Error reading bytes, read:", n, "error", err)
    } else {
        if string(p) != string(testBytes) {
            t.Fatal("Byte read does not match byte written",
                string(p) , string(testBytes))
        } else {
            log.Println("Small block write OK!")
        }
    }
}

func TestBlob_WriteLarge(t *testing.T) {

    var testBytes []byte = make([]byte, 2000)
    for i := range testBytes {
        testBytes[i] = byte(i)
    }

    var testPosition int64 = 0
    var testBlobSize int64 = int64(len(testBytes)) + testPosition

    blob1, err := client1.CreateBlob([]byte("test"), testBlobSize, ConsistencyLevelTwo)
    if err != nil {
        t.Fatal("Failed to open blob")
    }

    blob2, _ := client2.OpenBlob([]byte("test"), ConsistencyLevelTwo)
    client3.OpenBlob([]byte("test"), ConsistencyLevelTwo)

    if blob2.Size() != testBlobSize {
        t.Fatal("Blob size is wrong!", testBlobSize, blob2.Size())
    }

    log.Println("Writing bytes")
    if n, err := blob1.WriteAt(testBytes, testPosition); err != nil {
        t.Fatal("Error writing bytes, wrote:", n, "error", err)
    }
    log.Println("Reading bytes back")
    p := make([]byte, len(testBytes))
    if n, err := blob2.ReadAt(p, testPosition); err != nil {
        t.Fatal("Error reading bytes, read:", n, "error", err)
    } else {
        for i := range p {
            if p[i] != testBytes[i] {
                t.Fatal("Byte read does not match byte written")
            }
        }
        log.Println("Large block write OK!")
    }
}


func startTestCluster() (*Client, *Client, *Client, *Client, *Client) {
    client1, err := NewClient(
        "local.", "node1", "test", 127, "127.0.0.1", 9991)

    client2, err := NewClient(
        "local.", "node2", "test", 127, "127.0.0.1", 9992)

    client3, err := NewClient(
        "local.", "node3", "test", 127, "127.0.0.1", 9993)

    client4, err := NewClient(
        "local.", "node4", "test", 127, "127.0.0.1", 9994)

    client5, err := NewClient(
        "local.", "node5", "test", 127, "127.0.0.1", 9995)

    if err != nil {
        log.Fatal("Error creating client", err)
    }

    log.Println("Sleeping 11 seconds")
    time.Sleep(11 * time.Second)

    return client1, client2, client3, client4, client5
}
