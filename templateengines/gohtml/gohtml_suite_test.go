package gohtml_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGohtml(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gohtml Suite")
}
