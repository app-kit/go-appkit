package caches_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCaches(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Caches Suite")
}
