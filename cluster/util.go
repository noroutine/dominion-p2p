package cluster

/*

Checks if three given hash values are in clockwise order on the hash ring

On the ring, there could be following situations

Order           Clockwise

a, b, c         true
a, c, b         false
b, a, c         false
b, c, a         true
c, a, b         true
c, b, a         false

 */
func Clockwise(a, b, c[]byte) bool {
    bc := CompareHashes(b, c)

    if bc < 0 {
        ba := CompareHashes(b, a)
        ac := CompareHashes(a, c)

        return ! (ba < 0 && ac < 0)
    } else {
        ca := CompareHashes(c, a)
        ab := CompareHashes(a, b)
        return ca < 0 && ab < 0
    }
}

// 1   if a > b
// 0   if a == b
// -1  if a < b
func CompareHashes(a, b []byte) int {
    var i int
    for i = hash_byte_len - 1; i > 0 && a[i] == b[i]; i-- {
    }

    switch {
    case a[i] > b[i]: return 1
    case a[i] < b[i]: return  -1
    }

    return 0
}


