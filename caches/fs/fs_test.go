package fs_test

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/theduke/go-appkit/caches/fs"
	"github.com/theduke/go-appkit/caches/tests"
)

var _ = Describe("Redis", func() {
	tmpDir := path.Join(os.TempDir(), "appkit_caches_fs_test")
	cache, err := fs.New(tmpDir)
	if err != nil {
		panic(err)
	}

	It("Should create", func() {
		_, err := fs.New(tmpDir)
		Expect(err).ToNot(HaveOccurred())
	})

	tests.TestCache(cache)
})
