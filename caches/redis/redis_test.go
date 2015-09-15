package redis_test

import (

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/theduke/go-appkit/caches/redis"
	"github.com/theduke/go-appkit/caches/tests"
)

var _ = Describe("Redis", func() {

	var redisCache *redis.Redis
	config := redis.Config{
		Address: "localhost:9999",
	}
	redisCache, _ = redis.New(config)
	
	It("Should create", func() {
		var err error
		redisCache, err = redis.New(config)
		Expect(err).ToNot(HaveOccurred())
	})


	tests.TestCache(redisCache)
})
