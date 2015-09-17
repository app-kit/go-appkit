package fs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"path"
	"testing"
)

func TestFs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fs Suite")
}

var _ = BeforeSuite(func() {

})

var _ = AfterSuite(func() {
	os.RemoveAll(path.Join(os.TempDir(), "appkit_caches_fs_test"))
})
