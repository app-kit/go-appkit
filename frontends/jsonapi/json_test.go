package jsonapi_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/backends/memory"
	"github.com/theduke/go-dukedb/backends/tests"

	kit "github.com/app-kit/go-appkit"
	. "github.com/app-kit/go-appkit/frontends/jsonapi"
)

func buildBackend() db.Backend {
	b := memory.New()

	b.RegisterModel(&tests.TestModel{})
	b.RegisterModel(&tests.TestParent{})

	return b
}

var _ = Describe("Json", func() {

	backend := buildBackend()

	It("Shold parse errors", func() {
		js := `{
			"errors": [{
				"code": "err1", "message": "err1"
			}, {
				"code": "err2", "message": "err2"
			}]
		}
		`

		var data ApiData
		Expect(json.Unmarshal([]byte(js), &data)).ToNot(HaveOccurred())

		Expect(data.Errors[0].Code).To(Equal("err1"))
		Expect(data.Errors[0].Message).To(Equal("err1"))
		Expect(data.Errors[1].Code).To(Equal("err2"))
	})

	It("Should convert errors", func() {
		resp := &kit.AppResponse{
			Error: kit.AppError{Code: "err", Message: "errmsg"},
		}

		convertedResp := ConvertResponse(backend, resp)

		var data ApiData
		Expect(json.Unmarshal(convertedResp.GetRawData(), &data)).ToNot(HaveOccurred())

		Expect(data.Errors[0].Code).To(Equal("err"))
		Expect(data.Errors[0].Message).To(Equal("errmsg"))
	})

	It("Should convert internal error", func() {
		resp := &kit.AppResponse{
			Error: kit.AppError{Code: "err", Message: "errmsg"},
		}

		convertedResp := ConvertResponse(backend, resp)

		var data ApiData
		Expect(json.Unmarshal(convertedResp.GetRawData(), &data)).ToNot(HaveOccurred())

		Expect(data.Errors[0].Code).To(Equal("internal_server_error"))
	})

	It("Should convert metadata", func() {
		resp := &kit.AppResponse{
			Meta: map[string]interface{}{"metakey": "metaval"},
		}

		convertedResp := ConvertResponse(backend, resp)

		var data ApiData
		Expect(json.Unmarshal(convertedResp.GetRawData(), &data)).ToNot(HaveOccurred())

		Expect(data.Meta["metakey"]).To(Equal("metaval"))
	})

	It("Should .BuildModel()", func() {
		json := `{
			"data": {
				"type": "test_models",
				"id": "33",
				"attributes": {
					"strVal": "val",
					"intVal": 33
				}
			}
		}
		`
		rawModel, err := BuildModel(backend, "", []byte(json))
		Expect(err).ToNot(HaveOccurred())
		model := rawModel.(*tests.TestModel)
		Expect(model.StrVal).To(Equal("val"))
		Expect(model.IntVal).To(Equal(int64(33)))
		Expect(model.GetId()).To(Equal("33"))
	})

	It("Should convert single model response", func() {
		model := &tests.TestModel{
			Id:     33,
			StrVal: "val",
			IntVal: 33,
		}

		resp := &kit.AppResponse{
			Data: model,
		}

		newResp := ConvertResponse(backend, resp)

		var data ApiModelData
		Expect(json.Unmarshal(newResp.GetRawData(), &data)).ToNot(HaveOccurred())

		Expect(data.Data.Id).To(Equal("33"))
		Expect(data.Data.Attributes["strVal"]).To(Equal("val"))
		Expect(data.Data.Attributes["intVal"]).To(Equal(float64(33)))
	})

	It("Should convert multiple model response", func() {
		model1 := &tests.TestModel{
			Id:     1,
			StrVal: "val1",
			IntVal: 1,
		}

		model2 := &tests.TestModel{
			Id:     2,
			StrVal: "val2",
			IntVal: 2,
		}

		resp := &kit.AppResponse{
			Data: []kit.Model{model1, model2},
		}

		newResp := ConvertResponse(backend, resp)

		var data ApiModelsData
		Expect(json.Unmarshal(newResp.GetRawData(), &data)).ToNot(HaveOccurred())

		Expect(data.Data[0].Id).To(Equal("1"))
		Expect(data.Data[1].Id).To(Equal("2"))
	})

	It("Should convert relationships", func() {
		model := tests.NewTestParent(1, true)
		resp := &kit.AppResponse{
			Data: &model,
		}
		newResp := ConvertResponse(backend, resp)

		var data ApiModelData
		Expect(json.Unmarshal(newResp.GetRawData(), &data)).ToNot(HaveOccurred())

		modelData := data.Data
		Expect(modelData.Id).To(Equal(model.GetId()))
		Expect(modelData.GetRelation("child")[0].Id).To(Equal(model.Child.GetId()))
	})
})
