package tests

import (
	"time"
	"sort"

	"github.com/theduke/go-appkit/caches"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCache(cache caches.Cache) {
	It("Should .Clear() when emtpy", func() {
		Expect(cache.Clear()).ToNot(HaveOccurred())
		Expect(cache.Keys()).To(Equal([]string{}))
	})

	It("Should SetString()", func() {
		Expect(cache.SetString("test1", "testval", nil, nil)).ToNot(HaveOccurred())
		Expect(cache.GetString("test1")).To(Equal("testval"))
	})

	It("Should set string cache item with tags and expires", func() {
		item := &caches.StrItem{
			Key: "test2",
			Value: "testval",
			ExpiresAt: time.Now().Add(time.Second * 60),
			Tags: []string{"tag1", "tag2"},
		}
		Expect(cache.Set(item)).ToNot(HaveOccurred())

		ritem, err := cache.Get(item.Key)
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
		keys, err := cache.Keys()
		Expect(err).ToNot(HaveOccurred())

		sort.Sort(sort.StringSlice(keys))

		Expect(keys).To(Equal([]string{"test1", "test2"}))
	})

	It("Should .Clear()", func() {
		Expect(cache.Clear()).ToNot(HaveOccurred())
		Expect(cache.Keys()).To(Equal([]string{}))
	})
	

	It("Should .Delete()", func() {
		Expect(cache.SetString("deletekey", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(cache.Delete("deletekey")).ToNot(HaveOccurred())
		Expect(cache.GetString("deletekey")).To(Equal(""))
	})

	It("Should .Delete() multiple", func() {
		Expect(cache.Clear()).ToNot(HaveOccurred())

		Expect(cache.SetString("deletekey1", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(cache.SetString("deletekey2", "val", nil, nil)).ToNot(HaveOccurred())
		Expect(cache.SetString("deletekey3", "val", nil, nil)).ToNot(HaveOccurred())

		Expect(cache.Delete("deletekey1", "deletekey2", "deletekey3")).ToNot(HaveOccurred())
		Expect(cache.Keys()).To(Equal([]string{}))
	})

	It("Should .KeysByTags()", func() {
		Expect(cache.SetString("k1", "val", nil, []string{"tag1"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k2", "val", nil, []string{"tag2", "tag1", "tag3"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k3", "val", nil, []string{"tag2", "tag1"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k4", "val", nil, []string{"tag33", "tag3"})).ToNot(HaveOccurred())

		keys, err := cache.KeysByTags("tag1")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k2", "k3"}))

		keys, err = cache.KeysByTags("tag3")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k2", "k4"}))

		keys, err = cache.KeysByTags("tag33", "tag1")
		Expect(err).ToNot(HaveOccurred())
		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k2", "k3", "k4"}))
	})

	It("Should .ClearTag()", func() {
		Expect(cache.Clear()).ToNot(HaveOccurred())

		Expect(cache.SetString("k1", "val", nil, []string{"tag1"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k2", "val", nil, []string{"tag2", "tag1", "tag3"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k3", "val", nil, []string{"tag2", "tag1"})).ToNot(HaveOccurred())
		Expect(cache.SetString("k4", "val", nil, []string{"tag33", "tag3"})).ToNot(HaveOccurred())

		Expect(cache.ClearTag("tag3")).ToNot(HaveOccurred())

		keys, err := cache.Keys()
		Expect(err).ToNot(HaveOccurred())

		sort.Sort(sort.StringSlice(keys))
		Expect(keys).To(Equal([]string{"k1", "k3"}))
	})

	It("Should .Set() and .Get() MapItem", func() {
		item := &caches.MapItem{
			Value: map[string]interface{}{"key1": "val1", "key2": "val2"},
		}
		item.Key = "map1"

		Expect(cache.Set(item)).ToNot(HaveOccurred())
		Expect(cache.Get("map1", &caches.MapItem{})).To(Equal(item))
	})

}
