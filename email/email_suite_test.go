package email_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEmail(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Email Suite")
}
