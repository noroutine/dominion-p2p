package cluster

import (
    "testing"
    "fmt"
)

var (
    biggest_hash [16]byte = [16]byte {
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
    }

    big_hash [16]byte = [16]byte {
        0x00, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
        0xFF, 0xFF,
    }

    smallest_hash [16]byte = [16]byte {
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
    }

    small_hash [16]byte = [16]byte {
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0x00,
        0x00, 0xFF,
    }

    ordered [4][]byte = [4][]byte {
        smallest_hash[:],
        small_hash[:],
        big_hash[:],
        biggest_hash[:],
    }

)

func TestCompareHashes(t *testing.T) {
    hashes := ordered

    for i := range hashes {
        for j := range hashes {
            cmpResult := CompareHashes(hashes[i], hashes[j])
            if (i < j && cmpResult != -1) || (i > j && cmpResult != 1) || (i == j && cmpResult != 0) {
                t.Fail()
            }
        }
    }
}

func TestClockwise(t *testing.T) {
    hashes := ordered

    for i := range hashes {
        a, b, c := hashes[i], hashes[(i + 1) % len(hashes)], hashes[(i + 2) % len(hashes)]

        if ! Clockwise(a, b, c) {
            t.Error(fmt.Sprintf("failed clockwise at %d", i))
        }
    }

    for i := range hashes {
        a, b, c := hashes[(i + 1) % len(hashes)], hashes[i], hashes[(i + 2) % len(hashes)]

        if Clockwise(a, b, c) {
            t.Error(fmt.Sprintf("failed non-clockwise at %d", i))
        }
    }
}