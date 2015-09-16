package gomail_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGomail(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gomail Suite")
}
