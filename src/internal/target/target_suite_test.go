package target_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTargetProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Target Suite")
}
