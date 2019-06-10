package dbx

import (
	"fmt"
	"reflect"
	"strings"
)

type Field struct {
	Table string
	Name  string
}

type Searchable interface {
	Fields() []Field
	SelectFrom() string
	CriteriaAlias(string) (Field, error)
}

var (
	escapes string = "\"'\\;%_.\n\t\n\r\b"
)

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

func NormalizeField(fieldname string) (string, error) {
	fn := strings.ToLower(fieldname)
	if strings.ContainsAny(fn, escapes) {
		return "", fmt.Errorf("feildname %s contains escape character, reject", fieldname)
	}

	return fn, nil
}

func NormalCriteriaAlias(s Searchable, fieldname string) (Field, error) {
	fn, err := NormalizeField(fieldname)
	if err != nil {
		return Field{}, err
	}

	fds := s.Fields()
	for _, fd := range fds {
		if fd.Name == fn {
			return fd, nil
		}
	}

	return Field{}, fmt.Errorf("cannot find field %s", fieldname)
}
