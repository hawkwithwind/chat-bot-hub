package dbx

import (
	"reflect"
)

type Field struct {
	Table string
	Name  string
}

type Searchable interface {
	Fields() []Field
	SelectFrom() string
}

func GetFieldsFromStruct(tablename string, i interface{}) []Field {
	t := reflect.TypeOf(i).Elem()

	fs := []Field{}
	for i := 0; i < t.NumField(); i++ {
		if tag, ok := t.Field(i).Tag.Lookup("db"); ok {
			fs = append(fs, Field{tablename, tag})
		}        
    }

	return fs
}
