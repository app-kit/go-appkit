package main

import (
	"reflect"
	"log"
)

func main() {
	filters := map[string]interface{} {
		"strVal": "str",
		"intVal": 33,
	}

	reflFilters := reflect.ValueOf(filters)

	for _, field := range reflFilters.MapKeys() {
		name := field.String()

		val := reflFilters.MapIndex(field)
		kind := val.Elem().Kind()

		log.Printf("name: %v\n", name)
		log.Printf("type: %v\n", kind)
		log.Printf("val: %v\n\n", val.Interface())
	}
}
