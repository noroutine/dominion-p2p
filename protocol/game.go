package protocol
/**
    Implemented according to
    https://docs.google.com/document/d/1uY5v8RunCTJa-8RE7EHW3UyAnHtTjr3BXNh0qXlMZu4/edit#heading=h.j1mgskotdy4y
 */

type GameState struct {
    Copper, Silver, Gold int    // amount of copper, silver and gold coins
    V1, V3, V6 int              // amount of victory cards of each value
    Actions map[string]int      // amount of 10 types of action cards in the game
    Curse int                   // amount of curse cards
    Trash []string              // trash deck
}

type TransactionRecord struct {
    Id []byte                   // hash of current game transaction
    PrevId []byte               // reference to hash of previous game transaction
    Tx []byte                   // transaction type
    Signature []byte            // digital signature of (PrevId, PrevSignature, Id, Tx)
}

type TransactionIndex struct {
    // TBD
}

type Game struct {
    StartTx []byte              // hash of first transaction
    LastTx  []byte              // hash of last transaction seen by current client
    Index   TransactionIndex    // B-tree index of transactions
}

