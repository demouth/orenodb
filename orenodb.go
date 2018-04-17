package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tidwall/buntdb"
)

func main() {

	sigChan := make(chan os.Signal, 1)
	// Ignore all signals
	signal.Ignore()
	signal.Notify(sigChan, syscall.SIGINT)

	s := NewServer()

	s.Start()

	select {
	case sig := <-sigChan:
		switch sig {
		case syscall.SIGINT:
			log.Println("Start Server Shutdown")
			s.Stop()
			log.Println("End Server Shutdown")
		default:
			panic("unexpected signal has been received")
		}
	}
}

// Server ...
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	wg       *sync.WaitGroup
	db       *Database
	listener net.Listener
}

// NewServer ...
func NewServer() *Server {
	s := &Server{}
	return s
}

// Start ...
func (s *Server) Start() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	s.ctx = ctx
	s.cancel = cancel
	s.wg = &wg
	s.db = NewDatabase()
	s.db.Open()

	listener, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		panic(err)
	}
	s.listener = listener
	go s.startServer()
}

func (s *Server) startServer() {

	fmt.Println("Server is running at localhost:8888")
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("Error ", err)
			return
		}
		s.wg.Add(1)
		connection := NewConnection(s.ctx, s.wg, s.db)
		go connection.Start(conn)
	}
}

// Stop ...
func (s *Server) Stop() {
	s.cancel()
	s.wg.Wait()
	s.listener.Close()
	s.db.Close()
}

// Connection ...
type Connection struct {
	ctx context.Context
	wg  *sync.WaitGroup
	db  *Database
}

// NewConnection ...
func NewConnection(ctx context.Context, wg *sync.WaitGroup, db *Database) *Connection {
	c := &Connection{ctx, wg, db}
	return c
}

// Start ...
func (c *Connection) Start(conn net.Conn) {

	remoteAddr := conn.RemoteAddr()
	errCh := make(chan error, 1)
	defer func() {
		fmt.Println("Closing", remoteAddr)
		time.Sleep(1 * time.Second)
		conn.Close()
		c.wg.Done()
		fmt.Println("Closed", remoteAddr)
	}()
	go func() {
		fmt.Printf("Accept %v\n", remoteAddr)
		cp := NewCommandParser()
		for {
			buf := make([]byte, 4*1024)
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- err
				fmt.Println("Error ", err)
				return
			}

			if cp.Add(buf[:n]) == false {
				continue
			}
			fmt.Println(cp.commands)

			switch strings.ToUpper(cp.commands[0]) {
			case "GET":
				v := c.db.Read(cp.commands[1])
				conn.Write([]byte(TokenizeBulkString(v)))
			case "KEYS":
				v, _ := c.db.ReadKyes()
				conn.Write([]byte(TokenizeBulkStrings(v)))
			case "SET":
				c.db.Write(cp.commands[1], cp.commands[2])
				conn.Write([]byte(TokenizeSimpleString("OK")))
			case "PING":
				conn.Write([]byte(TokenizeSimpleString("PONG")))
			}
			cp.Reset()

		}
	}()
	select {
	case <-errCh:
	case <-c.ctx.Done():
	}
}

// CommandParser ...
//
// https://redis.io/topics/protocol
//
type CommandParser struct {
	s        []byte
	commands []string
}

// NewCommandParser ...
func NewCommandParser() *CommandParser {
	cp := &CommandParser{
		make([]byte, 0),
		make([]string, 0),
	}
	return cp
}

// Add ...
func (cp *CommandParser) Add(s []byte) bool {
	cp.s = append(cp.s, s...)
	parseSuccess, _, _ := cp.parse(0)
	return parseSuccess
}

// Reset ...
func (cp *CommandParser) Reset() {
	cp.s = []byte{}
	cp.commands = []string{}
}

func (cp *CommandParser) parse(seek int) (bool, []byte, int) {

	crlfLength := 2

	var commands []string
	seekEnd := strings.Index(string(cp.s[seek:]), "\r\n")

	if seekEnd < 0 {
		return false, nil, -1
	}

	seekEnd += seek

	typeStr := string(cp.s[seek])
	length, _ := strconv.Atoi(string(cp.s[seek+1 : seekEnd]))

	switch typeStr {
	case "*": //Array
		seek = seekEnd + crlfLength
		for length > 0 {
			ok, cmd, newSeek := cp.parse(seek)
			if ok == false {
				return false, nil, -1
			}
			commands = append(commands, string(cmd))
			seek = newSeek
			length--
		}
	case "$": //Bulk String
		if len(cp.s) < seekEnd+crlfLength+length+crlfLength {
			return false, nil, -1
		}
		command := cp.s[seekEnd+crlfLength : seekEnd+crlfLength+length]
		return true, command, seekEnd + crlfLength + length + crlfLength
	default:
		panic("Incorrect format")
	}

	cp.commands = commands
	return true, nil, seek + crlfLength
}

// CommandTokenizer ...
type CommandTokenizer struct {
	//TODO
}

// TokenizeBulkStrings ...
func TokenizeBulkStrings(in []string) string {
	out := "*" + strconv.Itoa(len(in)) + "\r\n"
	for _, v := range in {
		out += TokenizeBulkString(v)
	}
	return out
}

// TokenizeBulkString ...
func TokenizeBulkString(in string) string {
	out := "$" + strconv.Itoa(len(in)) + "\r\n" + in + "\r\n"
	return out
}

// TokenizeSimpleString ...
func TokenizeSimpleString(in string) string {
	out := "+" + in + "\r\n"
	return out
}

// Database ...
type Database struct {
	buntdb *buntdb.DB
}

// NewDatabase ...
func NewDatabase() *Database {
	db := &Database{}
	return db
}

// Open ...
func (db *Database) Open() {
	buntdb, err := buntdb.Open(":memory:")
	db.buntdb = buntdb
	if err != nil {
		panic(err)
	}
}

// Close ...
func (db *Database) Close() {
	db.buntdb.Close()
}

func (db *Database) Write(key, value string) {
	if err := db.buntdb.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(fmt.Sprintf("%s", key), fmt.Sprintf("%s", value), nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		panic(err)
	}
}

// Read ...
func (db *Database) Read(key string) string {
	var val string
	if err := db.buntdb.View(func(tx *buntdb.Tx) error {
		var err error
		val, err = tx.Get(key)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return val
}

// Indexes ...
func (db *Database) Indexes() []string {
	names, err := db.buntdb.Indexes()
	if err != nil {
		panic(err)
	}
	return names
}

// ReadKyes ...
func (db *Database) ReadKyes() ([]string, error) {

	results := []string{}
	err := db.buntdb.View(func(tx *buntdb.Tx) error {
		err2 := tx.Ascend("", func(key, value string) bool {
			fmt.Printf("key: %s, value: %s\n", key, value)
			results = append(results, key)
			return true
		})
		if err2 != nil {
			panic(err2)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return results, err

}

func openDb() *buntdb.DB {
	db, err := buntdb.Open("data.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.Update(func(tx *buntdb.Tx) error {
		for i := 0; i < 20; i++ {
			_, _, err := tx.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("planet:%d", i), nil)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
	if err := db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("key:13")
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	}); err != nil {
		panic(err)
	}

	return db
}
