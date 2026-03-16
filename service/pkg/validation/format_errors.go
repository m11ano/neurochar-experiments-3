package validation

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatErrors(err error, subjects ...any) []string {
	hints := make([]string, 0)

	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return hints
	}

	var root reflect.Value
	if len(subjects) > 0 && subjects[0] != nil {
		root = reflect.ValueOf(subjects[0])
		for root.Kind() == reflect.Ptr && !root.IsNil() {
			root = root.Elem()
		}
	}

	for _, fe := range ve {
		if root.IsValid() {
			if msg, ok := lookupErrMsgTag(root, fe); ok && msg != "" {
				hints = append(hints, msg)
				continue
			}
		}
		hints = append(hints, fe.Error())
	}

	return hints
}

func lookupErrMsgTag(root reflect.Value, fe validator.FieldError) (string, bool) {
	if !root.IsValid() || root.Kind() != reflect.Struct {
		return "", false
	}

	path := fe.StructNamespace()
	if path == "" {
		path = fe.StructField()
	}
	segments := strings.Split(path, ".")
	if len(segments) > 1 && segments[0] == root.Type().Name() {
		segments = segments[1:]
	}

	indexRe := regexp.MustCompile(`\[[0-9]+\]`)
	rt := root.Type()
	current := root
	var sf reflect.StructField
	found := false

	for _, seg := range segments {
		seg = indexRe.ReplaceAllString(seg, "")
		f, ok := findField(rt, seg)
		if !ok {
			return "", false
		}
		sf = f
		found = true

		if current.IsValid() && current.Kind() == reflect.Struct {
			fieldVal := current.FieldByIndex(sf.Index)
			for fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() {
				fieldVal = fieldVal.Elem()
			}
			current = fieldVal
			rt = current.Type()
		}
	}

	if !found {
		return "", false
	}
	if v, ok := sf.Tag.Lookup("errmsg"); ok {
		return v, true
	}
	return "", false
}

func findField(t reflect.Type, name string) (reflect.StructField, bool) {
	if f, ok := t.FieldByName(name); ok {
		return f, true
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			ft := f.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				if ff, ok := findField(ft, name); ok {
					return combineIndex(f, ff), true
				}
			}
		}
	}
	return reflect.StructField{}, false
}

func combineIndex(parent reflect.StructField, child reflect.StructField) reflect.StructField {
	idx := make([]int, 0, len(parent.Index)+len(child.Index))
	idx = append(idx, parent.Index...)
	idx = append(idx, child.Index...)
	child.Index = idx
	return child
}
