package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "flag"
    
    "github.com/noroutine/go-cli"
    "github.com/noroutine/dominion/protocol"
    "github.com/noroutine/dominion/cluster"
    "math/big"
)

const version = "0.0.7"
const description = "Dominion " + version

type Options struct {
    port int
    name string
    join string
    announce bool
}

func main() {

    opts := Options{}

    flag.IntVar(&opts.port, "port", cluster.DefaultPort, "client API port")
    flag.StringVar(&opts.name, "name", "", "name of the player")
    flag.StringVar(&opts.join, "join", "", "name of the group of the node")
    flag.Parse()

    if opts.port <= 0 || opts.port > 65535 {
        fmt.Printf("Invalid port: %v\n", opts.port)
        os.Exit(42)
    }

    opts.name = strings.TrimSpace(opts.name)
    if len(opts.name) == 0 {
        fmt.Printf("Must provide name, see --help\n")
        os.Exit(42)
    }

    opts.join = strings.TrimSpace(opts.join)
    if len(opts.join) == 0 {
        fmt.Printf("Must provide group, see --help\n")
        os.Exit(42)
    }

    client := protocol.NewClient(fmt.Sprintf(":%d", opts.port), opts.name)
    go client.Serve()

    repl := cli.New()
    repl.Description = description
    repl.Prompt = opts.name + "> "

    go func() {
        for s := range repl.Signals {
            if s == os.Interrupt {
                log.Printf("Interrupted")
                os.Exit(42)
            }
        }
    }()

    node := cluster.NewNode("local.", opts.name)
    node.Port = opts.port
    node.Group = &opts.join

    node.AnnouncePresence()
    node.StartDiscovery()

    cl, err := cluster.NewVia(node)
    if err != nil {
        log.Fatal(fmt.Sprintln("Cannot start cluster", err))
    }

    go func() {
        for {
            select {
            case peer := <- node.Joined:
                if *peer.Name == *node.Name {
                    // stupid but anyways: once I notice I joined, I connect to cluster :D
                    cl.Connect()
                }
                log.Println(*peer.Name, "joined")
            case peer := <- node.Left:
                log.Println(*peer.Name, "left")
            }
        }
    }()

    repl.EmptyHandler = func() {        
        fmt.Println("Feeling lost? Try 'help'")
        repl.EmptyHandler = nil
    }
    
    repl.Register("peers", func(args []string) {
        if node.Group == nil {
            fmt.Println("You are not a member of any group, use 'join'")
            return
        }

        if (! node.IsDiscoveryActive()) {
            node.DiscoverPeers()
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

        fmt.Printf("Your peers in group %s:\n", *node.Group)
        peers := cl.Peers()
        prev := peers[len(peers) - 1]

        for i, p := range cl.Peers() {
            prevHash, peerHash := prev.Hash(), p.Hash()
            diff := new(big.Int).Sub(new(big.Int).SetBytes(peerHash), new(big.Int).SetBytes(prevHash))
            if i == 0 {
                diff = diff.Add(diff, keyspace)
            }

            percent, _ := new(big.Float).Mul(new(big.Float).Quo(new(big.Float).SetInt(diff), new(big.Float).SetInt(keyspace)), big.NewFloat(100)).Float64()

            fmt.Printf("%s (%s:%d) %x (%.2f%% of keys)\n", *p.Name, p.GetAddrIPv4(), p.Port, peerHash, percent)

            prev = p
        }
    })

    repl.Register("groups", func(args []string) {
        if (! node.IsDiscoveryActive()) {
            node.DiscoverPeers()
        }

        for name, data := range node.Groups {
            fmt.Printf("%s (%d members)\n", name, data.SeenMembers)
        }
    })

    repl.Register("group", func(args []string) {
        var group string
        if (node.Group == nil) {
            group = "None"
        } else {
            group = *node.Group
        }
        fmt.Println(group)
    })

    repl.Register("join", func(args []string) {
        var group string
        if len(args) > 0 {
            group = args[0]
            fmt.Println("Your group is now", group)
            node.AnnounceGroup(&group)
        } else {
            fmt.Println("Provide group name, discover with 'groups")
        }
    })

    repl.Register("leave", func(args []string) {
        node.AnnounceGroup(nil)
        cl.Disconnect()
    })

    repl.Register("find", func(args []string) {
        if len(args) < 1 {
            fmt.Println("Usage: find <key>")
        }

        obj := cluster.StringObject{
            Data: &args[0],
        }

        primary, secondary := cl.HashNodes(obj.Hash())

        fmt.Printf("Key %s stored by peers:\n  primary  : %s\n  secondary: %s\n", args[0], *primary.Name, *secondary.Name)
    })

    repl.Register("store", func(args []string) {
        if len(args) < 2 {
            fmt.Println("Usage: store <key> <value>")
        }

        switch cl.Store([]byte(args[0]), []byte(args[1])) {
        case cluster.STORE_SUCCESS: fmt.Println("Success")
        case cluster.STORE_PARTIAL_SUCCESS: fmt.Println("Partial success")
        case cluster.STORE_ERROR: fmt.Println("Error")
        case cluster.STORE_FAILURE: fmt.Println("Failure")
        }
    })

    repl.Register("load", func(args []string) {
        if len(args) < 1 {
            fmt.Println("Usage: load <key>")
        }

        data, result := cl.Load([]byte(args[0]))

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
        }

        switch cl.Ping(args[0]) {
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
            client.PlayerID = name
            cl.Disconnect()
            node.AnnounceName(name)
        } else {
            fmt.Println(*node.Name)
        }
    })

    repl.Serve()
}
