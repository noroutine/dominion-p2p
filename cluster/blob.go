package cluster

import (
    "io"
    "fmt"
    "errors"
)

/* Thoughts

When the real write shall happen What happens on concurrent writes to the
same blob by different clients
    - locking mechanism
    - exclusive locks
    - optimistic locks

The same question goes for simple block writes, not only for blobs

how does visibility for readers happen
    - when do readers see the changes
    - how do they know something was updated

How do we know which block to flush and which not, for example in the case
when data crosees the block boundary, we have 1 block that is full, that can
be flushed immediately and 1 block which is partially filled, that we might
want to wait to flush

Internal buffer memory?

Using consistent hashing has some interesting simplifications as a
consequence, for example you don't have to think about how two nodes will
coordinate the access to data block, since hashing gives you the guarantee
you will access the block through the same node!

 */


const (
    BLOB_CREATE int = iota
    BLOB_OPEN
)

type BlobWindow struct {
    buffer []byte
    position int
    length int
}

type Blob struct {
    client *Client
    key []byte
    pos int64
    consistencyLevel ConsistencyLevel
    pageSize int64
    pages map[int][]byte
}

func (client *Client) OpenBlob(key []byte) *Blob {
    // load blob data

    return &Blob{
        client: client,
        key: key,
        pos: 0,
        consistencyLevel: ConsistencyLevelTwo,
        pageSize: BlockSize,
        pages: make(map[int][]byte),
    }
}

func (blob *Blob) GetPageIndex(offset int64) int64 {
    return offset / blob.pageSize
}

func (blob *Blob) GetPageKey(offset int64) []byte {
    return append(blob.key,
        fmt.Sprintf(".%08x", blob.GetPageIndex(offset)) ...)
}

func (blob *Blob) LoadPageAtPosition(position int64) ([]byte, int) {
    return blob.client.Load(
        blob.GetPageKey(position), blob.consistencyLevel)
}

func (blob *Blob) FlushPageAtPosition(position int64, page []byte) int {
    return blob.client.Store(
        blob.GetPageKey(position), page, blob.consistencyLevel)
}

func (blob *Blob) Seek(offset int64, whence int) (n int64, err error) {
    newPos := blob.pos
    switch whence {
    case 0:
        newPos = offset
    case 1:
        newPos += offset
    case 2:
        return blob.pos, errors.New("Not imeplemented")
    default:
        return blob.pos, errors.New("Unknown whence, shall be 0, 1 or 2")
    }

    if newPos < 0 {
        return blob.pos, errors.New(
            "Atteppt to position before the beginning of the blob")
    }

    blob.pos = newPos
    return newPos, nil
}

func (blob *Blob) Write(p []byte) (n int, err error) {
    return blob.WriteAt(p, blob.pos)
}

func (blob *Blob) WriteByte(c byte) error {
    if err := blob.WriteByteAt(c, blob.pos); err != nil {
        return err
    }
    blob.pos++
    return nil
}

func (blob *Blob) WriteByteAt(c byte, position int64) error {
    page, result := blob.LoadPageAtPosition(position)
    pageOffset := position % blob.pageSize

    switch result {
        case LOAD_SUCCESS, LOAD_PARTIAL_SUCCESS:
            page[pageOffset] = c
            flushResult := blob.FlushPageAtPosition(position, page)
            switch flushResult {
            case STORE_SUCCESS, STORE_PARTIAL_SUCCESS:
                return nil
            case STORE_ERROR:
                return errors.New("Page store error")
            case STORE_FAILURE:
                return errors.New("Page store failure")
            default:
                return errors.New("Page store unexpected error")
            }
    case LOAD_ERROR:
        return errors.New("Page load error")
    case LOAD_FAILURE:
        return errors.New("Page load failure")
    default:
        return errors.New("Page load unexpected error")
    }
}

func (blob *Blob) WriteAt(p []byte, off int64) (n int, err error) {
    for i, b := range p {
        if err := blob.WriteByteAt(b, off + int64(i)); err != nil {
            return i, err
        }
    }
    return len(p), nil
}

func (blob *Blob) WriteTo(w io.Writer) (n int64, err error) {
    return 0, nil
}

func (blob *Blob) Read(p []byte) (n int, err error) {
    return blob.ReadAt(p, blob.pos)
}

func (blob *Blob) ReadByte() (c byte, err error) {
    if c, err := blob.ReadByteAt(blob.pos); err != nil {
        return 0, err
    } else {
        blob.pos++
        return c, nil
    }
}

func (blob *Blob) ReadByteAt(position int64) (c byte, err error) {
    page, result := blob.LoadPageAtPosition(position)
    pageOffset := position % blob.pageSize
    switch result {
    case LOAD_SUCCESS, LOAD_PARTIAL_SUCCESS:
        return page[pageOffset], nil
    case LOAD_ERROR:
        return 0, errors.New("Page load error")
    case LOAD_FAILURE:
        return 0, errors.New("Page load failure")
    default:
        return 0, errors.New("Page load unexpecte error")
    }
}

func (blob *Blob) UnreadByte() error {
    _, err := blob.Seek(-1, 1)
    return err
}

func (blob *Blob) ReadAt(p []byte, off int64) (n int, err error) {
    for i := range p {
        if c, err := blob.ReadByteAt(off + int64(i)); err != nil {
            return i, err
        } else {
            p[i] = c
        }
    }
    return len(p), nil
}

func (blob *Blob) ReadFrom(r io.Reader) (n int64, err error) {
    return 0, nil
}

func (blob *Blob) Close() error {
    return nil
}
