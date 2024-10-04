# panic-collector

This SDK provides a wrapper to the [panicwrap]("https://github.com/bugsnag/panicwrap") library from [bugsnag]("https://github.com/bugsnag") and the [openpgp](https://"github.com/ProtonMail/go-crypto/openpgp") library from [ProtonMail](https://github.com/ProtonMail).
It features a gRPC-Client that can fetch a public key from a given gRPC-Endpoint, 
then encrypts the collected panic information and send them to the gRPC-Server,
with the matching Methods, 
according to the .proto file and the generated pgp public/private key on the server.

The usage of the SDK is as simple as that:

```go
func main() {
    config := &paniccollector.Config{
        GrpcServerAddr: "127.0.0.1:50051",
    }

    err := paniccollector.Initialize(config)
    if err != nil {
        fmt.Printf("Failed to initialize panic collector: %v\n", err)
        return
    }
    
    fmt.Println("Hello World! About to cause a panic...")
    
    panic("This is a test panic!")
}
```

Use your Endpoint address to configure the collector and initialize it.

Any panic will now get fully end to end encrypted submitted via gRPC to your Endpoint.
