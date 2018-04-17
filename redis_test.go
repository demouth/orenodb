package main

import (
	"strconv"
	"sync"
	"testing"

	"github.com/gomodule/redigo/redis"
)

// Redis benchmark
//
// Enable redis beforehand
func _TestRedisParallel(t *testing.T) {
	// s := NewServer()
	// s.Start()
	var wg sync.WaitGroup

	gl := 100
	rl := 1000
	for i := 0; i < gl; i++ {
		wg.Add(1)
		go func() {
			conn, _ := redis.Dial("tcp", "localhost:6379")
			for j := 0; j < rl; j++ {
				conn.Do("SET", "key"+strconv.Itoa(j), "value"+strconv.Itoa(j))
			}
			conn.Close()
			wg.Done()
		}()
	}

	wg.Wait()
	// s.Stop()
}
