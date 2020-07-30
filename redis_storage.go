package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

const (
	HASH_KEY        = "url_map"
	SUB_CHANNEL_DEL = "short_url:del"
)

type RedisStorage struct {
	//conn redis.Conn
	pool redis.Pool

	addr string
	//subConn redis.Conn
}

func NewRedisStorage(addr string, db int) (*RedisStorage, error) {
	pool := redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialDatabase(db), redis.DialConnectTimeout(time.Second*1), redis.DialWriteTimeout(time.Second*1), redis.DialReadTimeout(time.Second*1))
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     3,
		MaxActive:   1000,
		IdleTimeout: time.Second * 240,
		Wait:        true,
	}

	//conn, err := redis.Dial("tcp", addr, redis.DialDatabase(db), redis.DialReadTimeout(time.Second*10), redis.DialWriteTimeout(time.Second*10))
	//conn, err := redis.Dial("tcp", addr, redis.DialDatabase(db))
	//if err != nil {
	//	return nil, err
	//}

	//subConn, err := redis.Dial("tcp", addr)
	//if err != nil {
	//	return nil, err
	//}

	return &RedisStorage{
		//conn: conn,
		pool: pool,
		//subConn: subConn,
		addr: addr,
	}, nil
}

func (s *RedisStorage) getConn() redis.Conn {
	return s.pool.Get()
}

// 不存在时返回 error
func (s *RedisStorage) Get(key string) (string, error) {
	conn := s.getConn()
	defer conn.Close()

	value, err := redis.String(conn.Do("HGET", HASH_KEY, key))

	if err != nil && err == redis.ErrNil {
		return "", KeyNotFound
	}
	return value, err
}

// 当 key 不存在时才成功
func (s *RedisStorage) Set(key string, value string) error {
	conn := s.getConn()
	defer conn.Close()

	ret, err := redis.Int(conn.Do("HSETNX", HASH_KEY, key, value))
	if err != nil {
		return err
	}

	if ret == 0 {
		return KeyAlreadyExist
	}

	return nil
}

// 若 key 不存在也视为成功, 不返回错误
func (s *RedisStorage) Del(key string) error {
	conn := s.getConn()
	defer conn.Close()

	cnt, err := redis.Int(conn.Do("HDEL", HASH_KEY, key))
	if err != nil {
		return err
	}

	// 若不存在该 key, 那就无需同步删除消息
	if cnt == 0 {
		return nil
	}

	// 发布删除 key 的消息
	return s.publishShortDelMsg(key)
}

func (s *RedisStorage) Range(callback func(short, long string) bool) error {
	conn := s.getConn()
	defer conn.Close()

	var short string
	var long string
	cursor := 0

	for {
		ret, err := redis.Values(conn.Do("HSCAN", HASH_KEY, cursor))
		if err != nil {
			return err
		}
		cursor, _ = redis.Int(ret[0], nil)
		values, _ := redis.Values(ret[1], nil)

		for i := 0; i < len(values); i += 2 {
			short, _ = redis.String(values[i], nil)
			long, _ = redis.String(values[i+1], nil)
			goon := callback(short, long)
			if !goon {
				return nil
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

const MSG_DEL = "del"

func (s *RedisStorage) publishShortDelMsg(short string) error {
	conn := s.getConn()
	defer conn.Close()

	log.Tracef("send del msg, short: %s", short)
	_, err := conn.Do("PUBLISH", SUB_CHANNEL_DEL, fmt.Sprintf("%s@%s", MSG_DEL, short))
	if err != nil {
		log.Warn(err)
	}
	return err
}

func (s *RedisStorage) ReceiveSub(handleDelShort func(short string)) {
	var conn redis.Conn
	var err error
	var needReconn bool
	var psc redis.PubSubConn

	// 断线重连
START:
	for {
		needReconn = false
		if conn == nil {
			needReconn = true
		} else if err = conn.Err(); err != nil {
			log.Error(err)
			needReconn = true
		} else if _, err := conn.Do("ping"); err != nil {
			log.Error(err)
			needReconn = true
		}

		// 重新连接
		if needReconn {
			conn, err = redis.Dial("tcp", s.addr)
			if err != nil {
				//log.Error(err)
				time.Sleep(time.Second * 1)
				continue
			}
			psc = redis.PubSubConn{Conn: conn}
		}

		// 订阅移除 short 的消息
		err := psc.Subscribe(SUB_CHANNEL_DEL)
		if err != nil {
			log.Error(err)
			time.Sleep(time.Second * 1)
			continue
		}

		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				strs := strings.SplitN(string(v.Data), "@", 2)
				switch strs[0] {
				case MSG_DEL:
					log.Tracef("receive del msg, short: %s", strs[1])
					handleDelShort(strs[1])
				default:
					log.Warnf("unknown msg type %s - %s", strs[0], strs[1])
				}
			case redis.Subscription:
				log.Infof("subscribe %s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				log.WithField("err", v.Error()).Error("redis receive err, retry after 1sec")
				time.Sleep(time.Second * 1)
				continue START
			}
		}
	}

}
