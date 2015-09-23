package utils

import (
	kit "github.com/theduke/go-appkit"
)

func InterfaceToModelSlice(items []interface{}) []kit.Model {
	models := make([]kit.Model, 0)
	for _, item := range items {
		models = append(models, item.(kit.Model))
	}
	return models
}
