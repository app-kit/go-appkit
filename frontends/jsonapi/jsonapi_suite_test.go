package jsonapi_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJsonapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jsonapi Suite")
}
