package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "flag"
    
    "github.com/noroutine/dominion/protocol"
    "github.com/noroutine/dominion/cluster"
    "net"
)

const version = "0.0.7"
const description = "Dominion " + version

type Options struct {
    bind string
    port int
    partitions int
    name string
    join string
    announce bool
}

func main() {

    opts := Options{}
    flag.StringVar(&opts.bind, "bind", "127.0.0.1", "IP address to use")
    flag.IntVar(&opts.port, "port", cluster.DefaultPort, "client API port")
    flag.IntVar(&opts.partitions, "partitions", cluster.DefaultPartitions, "amount of storage partitions")
    flag.StringVar(&opts.name, "name", "", "name of the player")
    flag.StringVar(&opts.join, "join", "", "name of the group of the node")
    flag.Parse()

    if net.ParseIP(opts.bind) == nil {
        fmt.Printf("Invalid address: %v\n", opts.bind)
        os.Exit(42)
    }

    if opts.port <= 0 || opts.port > 65535 {
        fmt.Printf("Invalid port: %v\n", opts.port)
        os.Exit(42)
    }

    if opts.partitions < 1 {
        fmt.Printf("number of partitions must be at least 1, requested %d\n", opts.partitions)
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

    clusterClient, err := cluster.NewClient("local.", opts.name, opts.join, opts.partitions, opts.bind, opts.port);

    if err != nil {
        log.Fatal(fmt.Sprintln("Cannot start cluster", err))
    }

    httpClient := protocol.NewHttpClient(fmt.Sprintf(":%d", opts.port), clusterClient)
    go httpClient.Serve()

    replClient := protocol.NewReplClient(opts.name, description, clusterClient);
    replClient.Serve()

}
