package main

import (
	"github.com/gomodule/redigo/redis"
	"log"
	"testing"
)

func TestRedis(t *testing.T) {
	storage, err := NewRedisStorage(":6379", 1)
	if err != nil {
		log.Fatal(err)
	}

	str, err := redis.String(storage.conn.Do("HGET", "abc", "field"))
	t.Log(str)
	t.Log(err)

	t.Log(storage.conn.Do("HSETNX", "abc", "field", 123))
	t.Log(storage.conn.Do("HSETNX", "abc", "field", 123))

	t.Log(redis.String(storage.conn.Do("HGET", "abc", "field")))

	t.Log(storage.conn.Do("DEL", "abc"))
	t.Log(storage.conn.Do("DEL", "abc"))
}

func TestRedisStorage_Range(t *testing.T) {
	storage, err := NewRedisStorage(":6379", 1)
	if err != nil {
		log.Fatal(err)
	}

	err = storage.Range(func(short, long string) bool {
		log.Println(short, long)
		return true
	})
	if err != nil {
		log.Fatal(err)
	}
}
