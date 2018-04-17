package main

import (
	"strconv"
	"sync"
	"testing"

	"github.com/gomodule/redigo/redis"
)

func TestRedigo(t *testing.T) {
	s := NewServer()
	s.Start()

	conn, _ := redis.Dial("tcp", "localhost:8888")
	conn.Do("SET", "key", "value")
	val, err := redis.String(conn.Do("GET", "key"))

	if err != nil {
		t.Fatal(err)
	}
	if val != "value" {
		t.Fatal(val)
	}
	conn.Close()
	s.Stop()
}

func TestRedigoConcurrent(t *testing.T) {
	s := NewServer()
	s.Start()

	conn1, _ := redis.Dial("tcp", "localhost:8888")
	conn2, _ := redis.Dial("tcp", "localhost:8888")

	conn1.Do("SET", "key", "value")
	val1, _ := redis.String(conn1.Do("GET", "key"))
	val2, _ := redis.String(conn2.Do("GET", "key"))

	if val1 != val2 {
		t.Fatal(val1 + " != " + val2)
	}
	conn1.Close()
	conn2.Close()
	s.Stop()
}

func TestRedigoParallel(t *testing.T) {
	s := NewServer()
	s.Start()
	var wg sync.WaitGroup

	gl := 100
	rl := 1000
	for i := 0; i < gl; i++ {
		wg.Add(1)
		go func() {
			conn, _ := redis.Dial("tcp", "localhost:8888")
			for j := 0; j < rl; j++ {
				conn.Do("SET", "key"+strconv.Itoa(j), "value"+strconv.Itoa(j))
			}
			conn.Close()
			wg.Done()
		}()
	}

	wg.Wait()
	s.Stop()
}

func TestTokenizeBulkStrings(t *testing.T) {
	bs := []string{
		"1e879de6ffa81e5b39aaab80061e1925",
		"911e6988248110172213d24a1f78a8e0",
	}
	r := TokenizeBulkStrings(bs)
	if r != "*2\r\n$32\r\n1e879de6ffa81e5b39aaab80061e1925\r\n$32\r\n911e6988248110172213d24a1f78a8e0\r\n" {
		t.Fatal(r)
	}
}
func TestTokenizeBulkString(t *testing.T) {
	if TokenizeBulkString("test") != "$4\r\ntest\r\n" {
		t.Fatal()
	}
}
func TestTokenizeSimpleString(t *testing.T) {
	if TokenizeSimpleString("OK") != "+OK\r\n" {
		t.Fatal()
	}
}

func TestCommandParse(t *testing.T) {
	var cp *CommandParser
	cp = NewCommandParser()
	cp.Add([]byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"))
	if len(cp.commands) != 3 {
		t.Fatal()
	}
	if cp.commands[0] != "SET" {
		t.Fatal()
	}
	if cp.commands[1] != "key" {
		t.Fatal()
	}
	if cp.commands[2] != "value" {
		t.Fatal()
	}

	cp.Reset()
	cp.Add([]byte("*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"))
	if len(cp.commands) != 2 {
		t.Fatal()
	}
	if cp.commands[0] != "GET" {
		t.Fatal()
	}
	if cp.commands[1] != "key" {
		t.Fatal()
	}
}
func TestCommandParse2(t *testing.T) {
	var cp *CommandParser
	cp = NewCommandParser()
	cp.Add([]byte("*"))
	cp.Add([]byte("2\r"))
	cp.Add([]byte("\n$3\r\nGET"))
	cp.Add([]byte("\r\n"))
	cp.Add([]byte("$"))
	cp.Add([]byte("3"))
	cp.Add([]byte("\r"))
	cp.Add([]byte("\n"))
	cp.Add([]byte("k"))
	cp.Add([]byte("e"))
	cp.Add([]byte("y"))
	cp.Add([]byte("\r"))
	cp.Add([]byte("\n"))
	cp.Add([]byte(""))
	cp.Add([]byte(""))
	if len(cp.commands) != 2 {
		t.Fatal()
	}
	if cp.commands[0] != "GET" {
		t.Fatal()
	}
	if cp.commands[1] != "key" {
		t.Fatal()
	}
}
