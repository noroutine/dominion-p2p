package cluster

import "log"

type ConsistencyLevel byte

const (
    LEVEL_ZERO      ConsistencyLevel = iota     // any single node
    LEVEL_ONE                                   // +1 replica
    LEVEL_TWO                                   // +2 replicas
    LEVEL_THREE                                 // +3 replicas

    LEVEL_QUORUM    ConsistencyLevel = 0x7F     // N / 2 + 1 copies
    LEVEL_ALL       ConsistencyLevel = 0xFF     // N copies
)

func (c *Cluster) Copies(level ConsistencyLevel) int {
    switch c.AdjustedConsistencyLevel(level) {
    case LEVEL_ZERO:
        return 1
    case LEVEL_ONE:
        return 2
    case LEVEL_TWO:
        return 3
    case LEVEL_THREE:
        return 4
    case LEVEL_QUORUM:
        return c.Quorum()
    case LEVEL_ALL:
        return c.Size()
    default:
        return 1
    }
}

func (c *Cluster) AdjustedConsistencyLevel(level ConsistencyLevel) ConsistencyLevel {

    if level == LEVEL_QUORUM || level == LEVEL_ALL {
        return level
    }

    highest := level

    switch c.Size() {
    case 1:
        highest = LEVEL_ZERO
    case 2:
        highest = LEVEL_ONE
    case 3:
        highest = LEVEL_TWO
    case 4:
        highest = LEVEL_THREE
    default:
        highest = level
    }

    if highest < level {
        log.Println("Lowering consistency level due to insufficient cluster size to", highest)
        return highest
    } else {
        return level
    }
}
