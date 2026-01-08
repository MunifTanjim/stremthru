package redis

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	r "github.com/redis/go-redis/v9"
)

type redisConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

func parseRedisConnectionURI(uri string) (*redisConfig, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	config := redisConfig{
		Addr:     u.Host,
		Username: u.User.Username(),
		Password: "",
		DB:       0,
	}
	password, _ := u.User.Password()
	config.Password = password
	if db, err := strconv.Atoi(strings.TrimPrefix(u.Path, "/")); err == nil {
		config.DB = db
	}
	return &config, nil
}

var redis = func() *r.Client {
	if config.RedisURI == "" {
		return nil
	}

	rconf, err := parseRedisConnectionURI(config.RedisURI)
	if err != nil {
		panic(err)
	}

	redis := r.NewClient(&r.Options{
		Addr:     rconf.Addr,
		Username: rconf.Username,
		Password: rconf.Password,
		DB:       rconf.DB,
	})

	return redis
}()

func GetClient() *r.Client {
	return redis
}

func IsAvailable() bool {
	return redis != nil
}
