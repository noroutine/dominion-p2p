package cluster

import "log"

type ConsistencyLevel byte

const (
    ConsistencyLevelZero ConsistencyLevel = iota          // any single node
    ConsistencyLevelOne                                   // +1 replica
    ConsistencyLevelTwo                                   // +2 replicas
    ConsistencyLevelThree                                 // +3 replicas

    ConsistencyLevelQuorum ConsistencyLevel = 0x7F        // N / 2 + 1 copies
    ConsistencyLevelAll ConsistencyLevel = 0xFF           // N copies
)

func (c *Cluster) Copies(level ConsistencyLevel) int {
    switch c.AdjustedConsistencyLevel(level) {
    case ConsistencyLevelZero:
        return 1
    case ConsistencyLevelOne:
        return 2
    case ConsistencyLevelTwo:
        return 3
    case ConsistencyLevelThree:
        return 4
    case ConsistencyLevelQuorum:
        return c.Quorum()
    case ConsistencyLevelAll:
        return c.Size()
    default:
        return 1
    }
}

func (c *Cluster) AdjustedConsistencyLevel(level ConsistencyLevel) ConsistencyLevel {

    if level == ConsistencyLevelQuorum || level == ConsistencyLevelAll {
        return level
    }

    highest := level

    switch c.Size() {
    case 1:
        highest = ConsistencyLevelZero
    case 2:
        highest = ConsistencyLevelOne
    case 3:
        highest = ConsistencyLevelTwo
    case 4:
        highest = ConsistencyLevelThree
    default:
        highest = level
    }

    if highest < level {
        log.Println("Lowering consistency level due to insufficient cluster size to", highest)
        return highest
    }

    return level
}
