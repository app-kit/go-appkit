package raymond_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/app-kit/go-appkit/templateengines/raymond"
)

var tpl1 string = `Test {{var}}`

var _ = Describe("Engine", func() {
	var engine *Engine

	BeforeEach(func() {
		engine = New()
	})

	It("Should .Build()", func() {
		t, err := engine.Build("tpl1", tpl1)
		Expect(err).ToNot(HaveOccurred())
		Expect(t).ToNot(BeNil())
	})

	It("Should .GetTemplate()", func() {
		t, _ := engine.Build("tpl1", tpl1)
		Expect(engine.GetTemplate("tpl1")).To(Equal(t))
	})

	It("Should .BuildAndRender()", func() {
		output, err := engine.BuildAndRender("tpl1", tpl1, map[string]interface{}{"var": "test"})
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(Equal([]byte("Test test")))
	})

	It("Should .Render()", func() {
		t, err := engine.Build("tpl1", tpl1)
		Expect(err).ToNot(HaveOccurred())
		Expect(t).ToNot(BeNil())

		output, err := engine.Render("tpl1", map[string]interface{}{"var": "test"})
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(Equal([]byte("Test test")))
	})
})
