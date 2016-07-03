package protocol

import (
    "github.com/noroutine/go-cli"
    "os"
    "log"
    "fmt"
    "github.com/noroutine/dominion/cluster"
    "math/big"
    "strings"
)

type ReplClient struct {
    Name string
    repl *cli.REPL
    client *cluster.Client
}

func NewReplClient(name string, description string, clusterClient *cluster.Client) *ReplClient {

    repl := cli.New()
    repl.Description = description
    repl.Prompt = name + "> "

    repl.EmptyHandler = func() {
        fmt.Println("Feeling lost? Try 'help'")
        repl.EmptyHandler = nil
    }

    go func() {
        for s := range repl.Signals {
            if s == os.Interrupt {
                log.Printf("Interrupted")
                os.Exit(42)
            }
        }
    }()

    repl.Register("nodes", func(args []string) {

        if ! clusterClient.IsMember() {
            fmt.Println("You are not a member of any group, use 'join'")
            return
        }

        fmt.Printf("Nodes in group %s:\n", clusterClient.GetGroup())
        for _, p := range clusterClient.DiscoverPeers() {
            fmt.Printf("%-20s (%s:%d)\n", *p.Name, p.AddrIPv4, p.Port)
        }
    })

    repl.Register("partitions", func(args []string) {
        if ! clusterClient.IsMember() {
            fmt.Println("You are not a member of any group, use 'join'")
            return
        }

        keyspace :=  new(big.Int).SetBytes([]byte {
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
            0xFF, 0xFF,
        })

        fmt.Printf("Partitions in group %s:\n", clusterClient.GetGroup())
        partitions := clusterClient.Partitions()
        prev := partitions[len(partitions) - 1]

        byPeer := make(map[string]float64)
        for i, p := range partitions {
            prevHash, partitionHash := prev.Hash(), p.Hash()
            diff := new(big.Int).Sub(new(big.Int).SetBytes(partitionHash), new(big.Int).SetBytes(prevHash))
            if i == 0 {
                diff = diff.Add(diff, keyspace)
            }

            percent, _ := new(big.Float).Mul(new(big.Float).Quo(new(big.Float).SetInt(diff), new(big.Float).SetInt(keyspace)), big.NewFloat(100)).Float64()

            // fmt.Printf("%-20s %x\t(%.2f%% of keys)\n", fmt.Sprintf("%s.%d", *p.Peer.Name, p.Partition), partitionHash, percent)

            byPeer[*p.Peer.Name] = byPeer[*p.Peer.Name] + percent

            prev = p
        }

        for _, peer := range clusterClient.DiscoverPeers() {
            fmt.Printf("%-20s %d\t(%.2f%% of keys)\n", *peer.Name, peer.Partitions, byPeer[*peer.Name])
        }
    })

    repl.Register("groups", func(args []string) {
        for name, data := range clusterClient.DiscoverGroups() {
            fmt.Printf("%s (%d members)\n", name, data.SeenMembers)
        }
    })

    repl.Register("group", func(args []string) {
        var group string
        if (clusterClient.IsMember()) {
            group = clusterClient.GetGroup()
        } else {
            group = "None"
        }
        fmt.Println(group)
    })

    repl.Register("join", func(args []string) {
        var group string
        if len(args) > 0 {
            group = args[0]
            fmt.Println("Your group is now", group)
            clusterClient.Join(group)
        } else {
            fmt.Println("Provide group name, discover with 'groups")
        }
    })

    repl.Register("leave", func(args []string) {
        clusterClient.Leave()
    })

    repl.Register("find", func(args []string) {
        if len(args) < 1 {
            fmt.Println("Usage: find <key>")
            return
        }

        obj := cluster.StringObject{
            Data: &args[0],
        }

        nodes := clusterClient.KeyNodes(obj.Hash(), cluster.ConsistencyLevelTwo)
        primary, secondary := nodes[0], nodes[1]
        fmt.Printf("Key %s stored by peers:\n  primary  : %s\n  secondary: %s\n", args[0], *primary.Name, *secondary.Name)
    })

    repl.Register("store", func(args []string) {
        if len(args) < 2 {
            fmt.Println("Usage: store <key> <value>")
            return
        }

        switch clusterClient.Store([]byte(args[0]), []byte(args[1]), cluster.ConsistencyLevelTwo) {
        case cluster.STORE_SUCCESS: fmt.Println("Success")
        case cluster.STORE_PARTIAL_SUCCESS: fmt.Println("Partial success")
        case cluster.STORE_ERROR: fmt.Println("Error")
        case cluster.STORE_FAILURE: fmt.Println("Failure")
        }
    })

    repl.Register("load", func(args []string) {
        if len(args) < 1 {
            fmt.Println("Usage: load <key>")
            return
        }

        data, result := clusterClient.Load([]byte(args[0]), cluster.ConsistencyLevelTwo)

        switch result {
        case cluster.LOAD_SUCCESS: fmt.Println("Success:", string(data))
        case cluster.LOAD_PARTIAL_SUCCESS: fmt.Println("Partial success:", string(data))
        case cluster.LOAD_ERROR: fmt.Println("Error")
        case cluster.LOAD_FAILURE: fmt.Println("Failure")
        }
    })

    repl.Register("ping", func(args []string) {
        if len(args) < 1 {
            fmt.Println("Usage: ping <peer>")
            return
        }

        switch clusterClient.Ping(args[0]) {
        case cluster.PING_SUCCESS: fmt.Println("Success")
        case cluster.PING_ERROR: fmt.Println("Error")
        case cluster.PING_TIMEOUT: fmt.Println("Timeout")
        }
    })

    repl.Register("help", func(args []string) {
        fmt.Printf("Commands: %s\n", strings.Join(repl.GetKnownCommands(), ", "))
    })

    repl.Register("name", func(args []string) {
        if len(args) > 0 {
            name := args[0]
            fmt.Println("You are now", name)
            repl.Prompt = name + "> "
            clusterClient.SetName(name)
        } else {
            fmt.Println(clusterClient.GetName())
        }
    })

    return &ReplClient{
        Name: name,
        repl: repl,
        client: clusterClient,
    }
}

func (replCient *ReplClient) Serve() {
    replCient.repl.Serve();
}
