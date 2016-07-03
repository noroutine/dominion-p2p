package protocol

import (
    "net/http"
    "log"
    "fmt"
    "html"
    "github.com/noroutine/dominion/cluster"
    "bytes"
)

type HttpClient struct {
    Address string
    node    *cluster.Node
    cl      *cluster.Cluster
}

func NewHttpClient(address string, client *cluster.Client) *HttpClient {
    return &HttpClient{
        Address:  address,
        node: client.Node,
        cl: client.Cluster,
    }
}

func (client *HttpClient) Serve() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        log.Printf("GET %s", html.EscapeString(r.URL.Path))
        fmt.Fprintf(w, "Hello from %s\n", html.EscapeString(*client.node.Name))
    })

    http.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
        log.Printf("GET %s", html.EscapeString(r.URL.Path))
        r.ParseForm()
        data, result := client.cl.Load([]byte(r.Form.Get("key")), cluster.ConsistencyLevelTwo)

        switch result {
        case cluster.LOAD_SUCCESS:
            fmt.Println("Success")
            fmt.Fprint(w, string(data))
        case cluster.LOAD_PARTIAL_SUCCESS:
            fmt.Println("Partial success")
            fmt.Fprint(w, string(data))
        case cluster.LOAD_ERROR: fmt.Println("Error")
            http.Error(w, "Error", http.StatusInternalServerError)
            return
        case cluster.LOAD_FAILURE: fmt.Println("Failure")
            http.Error(w, "Failure", http.StatusInternalServerError)
        }
    })

    http.HandleFunc("/store", func(w http.ResponseWriter, r *http.Request) {

        if r.Method != "POST" {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        log.Printf("POST %s", html.EscapeString(r.URL.Path))
        mp, e := r.MultipartReader()

        if e != nil {
            http.Error(w, "Not multipart/form-data", http.StatusNotAcceptable)
            return
        }

        part, e := mp.NextPart()

        if e != nil {
            http.Error(w, "No parts", http.StatusBadRequest)
            return
        }

        // read it
        key := part.FileName()
        var buf bytes.Buffer
        for buf1 := make([]byte, 1024);; {
            n, e := part.Read(buf1)
            if e != nil {
                break
            }

            if buf.Len() < cluster.MaxLoadLength {
                buf.Write(buf1[:n])
            } else {
                http.Error(w, "Too large", http.StatusRequestEntityTooLarge)
                return
            }
        }

        switch client.cl.Store([]byte(key), buf.Bytes(), cluster.ConsistencyLevelTwo) {
        case cluster.STORE_SUCCESS: fmt.Println("Success")
        case cluster.STORE_PARTIAL_SUCCESS: fmt.Println("Partial success")
        case cluster.STORE_ERROR:
            fmt.Println("Error")
            http.Error(w, "Error", http.StatusInternalServerError)
            return
        case cluster.STORE_FAILURE:
            fmt.Println("Failure")
            http.Error(w, "Failure", http.StatusInternalServerError)
            return
        }
    })

    log.Fatal(http.ListenAndServe(client.Address, nil))
}