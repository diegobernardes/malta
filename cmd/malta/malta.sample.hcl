transport {
  http {
    address = "0.0.0.0"
    port    = 8080
  }
}

external {
  etcd {
    embed                  = true
    data                   = "/tmp/malta/etcd"
    initialization-timeout = "10s"
  }

  etcd {
    dial-timeout    = "2s"
    request-timeout = "10s"
    endpoints       = ["0.0.0.0:2379"]
  }
}
