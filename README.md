<p align="center">
  <img src="logo2.png" width="80">
  <br>
  <img src="logo.png" width="200">

  <p align="center">OrenoDB is dataBase server implementation for learning. OrenoDB means MyDB in Japanese.</p>
</p>

----------------------

## Features

- In-memory Key-Value database
- Use redis protocol

## Build and Run

Start up at port 8888.

```sh
$ go build
$ ./orenodb
Server is running at localhost:8888
```

## Connection from client

Implementation using golang and redis client library.

```go
package main

import (
	"github.com/gomodule/redigo/redis"
)

func main() {
	conn, _ := redis.Dial("tcp", "localhost:8888")
	conn.Do("SET", "key", "value")
	val, err := redis.String(conn.Do("GET", "key"))

	if err != nil {
		t.Fatal(err)
	}
	if val != "value" {
		t.Fatal(val)
	}
}
```

Use telnet command.

```
$ telnet localhost 8888
Trying ::1...
telnet: connect to address ::1: Connection refused
Trying 127.0.0.1...
Connected to localhost.
Escape character is '^]'.
*3
$3
SET
$3
key
$5
value
+OK
```