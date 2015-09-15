package redis_test

import (
	"time"
	"sort"

	. "github.com/theduke/go-appkit/caches/redis"
	"github.com/theduke/go-appkit/caches"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis", func() {

	var redis *Redis
	config := Config{
		Address: "localhost:9999",
	}

	BeforeEach(func() {
		var err error
		redis, err = New(config)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should .Clear() when emtpy", func() {
		Expect(redis.Clear()).ToNot(HaveOccurred())
		Expect(redis.Keys()).To(Equal([]string{}))
	})

	It("Should SetString()", func() {
		Expect(redis.SetString("test1", "testval", nil, nil)).ToNot(HaveOccurred())
		Expect(redis.GetString("test1")).To(Equal("testval"))
	})

	It("Should set string cache item with tags and expires", func() {
		item := &caches.StrItem{
			Key: "test2",
			Value: "testval",
			ExpiresAt: time.Now().Add(time.Second * 60),
			Tags: []string{"tag1", "tag2"},
		}
		Expect(redis.Set(item)).ToNot(HaveOccurred())

		ritem, err := redis.Get(item.Key)
		Expect(err).ToNot(HaveOccurred())
		Expect(ritem).ToNot(BeNil())

		// Since ExpiresAt is converted to and from a second ttl,
		// it wont be exactly equal.
		// Set to nil and do a custom comparison for expires at.
		expiresDelta := int(item.GetExpiresAt().Sub(item.ExpiresAt).Seconds())

		t := time.Time{}
		item.SetExpiresAt(t)
		ritem.SetExpiresAt(t)
		Expect(ritem).To(Equal(item))

		Expect(expiresDelta).To(BeNumerically("<", 5))
	})

	It("Should .Keys()", func() {
		keys, err := redis.Keys()
		Expect(err).ToNot(HaveOccurred())

		sort.Sort(sort.StringSlice(keys))

		Expect(keys).To(Equal([]string{"test1", "test2"}))
	})

	It("Should .Clear()", func() {
		Expect(redis.Clear()).ToNot(HaveOccurred())
		Expect(redis.Keys()).To(Equal([]string{}))
	})
	

	It("Should .Delete()", func() {
		Expect(redis.SetString("deletekey", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(redis.Delete("deletekey")).ToNot(HaveOccurred())
		Expect(redis.GetString("deletekey")).To(Equal(""))
	})

	It("Should .Delete() multiple", func() {
		Expect(redis.Clear()).ToNot(HaveOccurred())

		Expect(redis.SetString("deletekey1", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(redis.SetString("deletekey2", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(redis.SetString("deletekey3", "val", nil, nil)).ToNot(HaveOccurred())

		Expect(redis.Delete("deletekey1", "deletekey2", "deletekey3")).ToNot(HaveOccurred())
		Expect(redis.Keys()).To(Equal([]string{}))
	})

	It("Should .KeysByTags()", func() {
		Expect(redis.SetString("k1", "val", nil, []string{"tag1"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k2", "val", nil, []string{"tag2", "tag1", "tag3"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k3", "val", nil, []string{"tag2", "tag1"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k4", "val", nil, []string{"tag33", "tag3"})).ToNot(HaveOccurred())

		keys, err := redis.KeysByTags("tag1")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k2", "k3"}))

		keys, err = redis.KeysByTags("tag3")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k2", "k4"}))

		keys, err = redis.KeysByTags("tag33", "tag1")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k2", "k3", "k4"}))
	})

	It("Should .ClearTag()", func() {
		Expect(redis.Clear()).ToNot(HaveOccurred())

		Expect(redis.SetString("k1", "val", nil, []string{"tag1"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k2", "val", nil, []string{"tag2", "tag1", "tag3"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k3", "val", nil, []string{"tag2", "tag1"})).ToNot(HaveOccurred())
		Expect(redis.SetString("k4", "val", nil, []string{"tag33", "tag3"})).ToNot(HaveOccurred())

		Expect(redis.ClearTag("tag3")).ToNot(HaveOccurred())

		keys, err := redis.Keys()
		Expect(err).ToNot(HaveOccurred())

		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k3"}))
	})
})
