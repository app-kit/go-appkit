package redis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"os"
	"os/exec"
	"path"
	"time"
)

func TestRedis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redis Suite")
}

var serverCmd *exec.Cmd
var tmpDir string
var finishedChannel chan bool

var _ = BeforeSuite(func() {
	tmpDir = path.Join(os.TempDir(), "appkit_cache_redis_test")
	err := os.MkdirAll(tmpDir, 0777)
	Expect(err).ToNot(HaveOccurred())

	args := []string{
		"--port", 
		"9999", 

		"--pidfile",
		path.Join(tmpDir, "redis.pid"),

		"--dir",
		tmpDir,
	}
	serverCmd = exec.Command("redis-server", args...) 

  err = serverCmd.Start()
  Expect(err).NotTo(HaveOccurred())

  // Give the server some time to start.
  time.Sleep(time.Second * 1)
})

var _ = AfterSuite(func() {
	serverCmd.Process.Kill()
	os.RemoveAll(tmpDir)  
})
