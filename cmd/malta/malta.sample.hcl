transport {
  http {
    address = "0.0.0.0"
    port    = 8080
  }
}

database {
  sqlite3 {
    file                 = "malta.sqlite3"
    max-open-connections = 100
    max-idle-connections = 100
    connection-lifetime  = "2m"
  }
}
