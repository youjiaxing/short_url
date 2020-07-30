package main

import (
	"errors"
	"fmt"
	"github.com/orcaman/concurrent-map"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	MaxAttempt = 20
)

var (
	KeyNotFound     = errors.New("key not found")
	KeyAlreadyExist = errors.New("key already exist")
)

// 短链生成器
type ShortUrlGenerator interface {
	gen(length int) string
}

// 链接存储库
type UrlRepository struct {
	cmap      cmap.ConcurrentMap // 并发安全的 map
	redis     *RedisStorage      // Redis 存储
	generator ShortUrlGenerator  // 短链生成器

	enablePersist bool // 开启持久化
	shortLen      int  // 生成的短链的长度
	minLongUrlLen int  // 最短的长链的长度
}

func NewUrlRepository(shortLen int, redisAddr string) *UrlRepository {
	var enablePersist bool
	var storage *RedisStorage
	var err error

	if len(redisAddr) != 0 {
		enablePersist = true
		storage, err = NewRedisStorage(redisAddr, 1)
		if err != nil {
			log.Fatal(err)
		}
	}

	repos :=  &UrlRepository{
		cmap:      cmap.New(),
		redis:     storage,
		generator: SimpleGen,

		shortLen:      shortLen,
		enablePersist: enablePersist,
	}

	go storage.ReceiveSub(repos.handleDelShort)

	// 初始不载入所有 short url
	//err = repos.load()
	//if err != nil {
	//	log.Fatal(err)
	//}
	return repos
}

// 不存在时返回错误
func (r *UrlRepository) Get(short string) (string, error) {
	// 读取本地缓存的值
	v, ok := r.cmap.Get(short)
	if ok {
		return v.(string), nil
	}

	var long string
	var err = KeyNotFound

	// 从 redis 读取
	if r.enablePersist {
		long, err = r.redis.Get(short)
		if err != nil {
			if err != KeyNotFound {
				log.Warn("error in Get: ", err)
			}
			return "", err
		}

		// 缓存读取的值
		r.cmap.Set(short, long)
	}

	return long, err
}

// short 存在时返回错误
func (r *UrlRepository) set(short string, long string) error {
	ok := r.cmap.SetIfAbsent(short, long)
	if !ok {
		return KeyAlreadyExist
	}

	// 持久化
	err := r.persist(short, long)
	if err != nil {
		return KeyAlreadyExist

		// 失败则 rollback
		r.cmap.Remove(short)
		return err
	}

	return nil
}

// 移除短链
func (r *UrlRepository) Delete(short string) error {
	var err error

	r.cmap.Remove(short)
	if r.enablePersist {
		err = r.redis.Del(short)

		//TODO 若开启多实例, 需要同步各个实例间的数据
	}
	return err
}

// 生成新的短链
func (r *UrlRepository) Put(long string) (string, error) {
	var short string
	for i := 0; i < MaxAttempt; i++ {
		// 随机生成短链
		short = r.generator.gen(r.shortLen)

		// 本地写入短链
		err := r.set(short, long)
		if err != nil {
			if err == KeyAlreadyExist {
				continue
			}
			return "", err
		}
		return short, nil
	}
	return "", fmt.Errorf("generate short fail after tried %d times", MaxAttempt)
}

func (r *UrlRepository) load() error {
	if !r.enablePersist {
		return nil
	}

	var n int
	err := r.redis.Range(func(short, long string) bool {
		r.cmap.Set(short, long)
		n++
		return true
	})

	log.Println("load from redis: ", n)
	return err
}

func (r *UrlRepository) persist(short, long string) error {
	if !r.enablePersist {
		return nil
	}
	return r.redis.Set(short, long)
}

// 解析并验证长 url
func (r *UrlRepository) parseLongUrl(long string) (string, error) {
	if len(long) <= 3 {
		return "", errors.New("invalid long url")
	}

	if !strings.HasPrefix(long, "http") {
		long = "http://" + long
	}

	return long, nil
}

func (r *UrlRepository) handleDelShort(short string)  {
	r.cmap.Remove(short)
}