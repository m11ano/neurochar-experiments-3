package dbhelper

import (
	"fmt"
	"reflect"
	"sync"
)

type ConverterFunc func(src interface{}) (interface{}, error)

var convertersMu sync.RWMutex

var converters = make(map[string]ConverterFunc)

func converterKey(srcType, dstType reflect.Type) string {
	return srcType.String() + "->" + dstType.String()
}

func RegisterBidirectionalConverter(dbType, domainType reflect.Type,
	dbToDomain ConverterFunc, domainToDB ConverterFunc,
) {
	convertersMu.Lock()
	defer convertersMu.Unlock()
	converters[converterKey(dbType, domainType)] = dbToDomain
	converters[converterKey(domainType, dbType)] = domainToDB
}

func getConverter(srcType, dstType reflect.Type) (ConverterFunc, bool) {
	convertersMu.RLock()
	defer convertersMu.RUnlock()
	conv, ok := converters[converterKey(srcType, dstType)]
	return conv, ok
}

func safeConvert(conv ConverterFunc, src interface{}) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in converter: %v", r)
		}
	}()
	result, err = conv(src)
	return
}

func ConvertDBToDomain(dbRecord, domainModel interface{}) error {
	dbVal := reflect.ValueOf(dbRecord)
	if dbVal.Kind() == reflect.Ptr {
		dbVal = dbVal.Elem()
	}
	if dbVal.Kind() != reflect.Struct {
		return fmt.Errorf("dbRecord must be a struct or pointer to a struct")
	}

	dVal := reflect.ValueOf(domainModel)
	if dVal.Kind() != reflect.Ptr || dVal.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("domainModel must be a pointer to a struct")
	}
	dVal = dVal.Elem()
	dType := dVal.Type()

	dbType := dbVal.Type()
	for i := 0; i < dbType.NumField(); i++ {
		dbField := dbType.Field(i)
		dbFieldValue := dbVal.Field(i)
		if !dbFieldValue.IsValid() {
			continue
		}

		fieldName := dbField.Name

		dField, found := dType.FieldByName(fieldName)
		if !found {
			continue
		}

		dFieldVal := dVal.FieldByName(dField.Name)
		if !dFieldVal.CanSet() {
			continue
		}

		srcType := dbFieldValue.Type()
		dstType := dFieldVal.Type()

		if conv, ok := getConverter(srcType, dstType); ok {
			converted, err := safeConvert(conv, dbFieldValue.Interface())
			if err != nil {
				return fmt.Errorf("converter error for field %s: %v", dbField.Name, err)
			}
			convVal := reflect.ValueOf(converted)
			if !convVal.Type().AssignableTo(dstType) {
				return fmt.Errorf("converted value for field %s is not assignable to type %s", dbField.Name, dstType)
			}
			dFieldVal.Set(convVal)
			continue
		}

		if srcType.AssignableTo(dstType) {
			dFieldVal.Set(dbFieldValue)
			continue
		}

		if srcType.ConvertibleTo(dstType) {
			dFieldVal.Set(dbFieldValue.Convert(dstType))
			continue
		}

		return fmt.Errorf("no converter registered for field %s: %s -> %s", dbField.Name, srcType, dstType)
	}

	return nil
}

func StructToDBMap(src, dbSchema interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	schemaVal := reflect.ValueOf(dbSchema)
	if schemaVal.Kind() == reflect.Ptr {
		schemaVal = schemaVal.Elem()
	}
	if schemaVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("dbSchema must be a struct or pointer to a struct")
	}
	schemaType := schemaVal.Type()

	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("src must be a struct or pointer to a struct")
	}
	srcType := srcVal.Type()

	for i := 0; i < schemaType.NumField(); i++ {
		schemaField := schemaType.Field(i)
		key := schemaField.Tag.Get("db")
		if key == "" {
			key = schemaField.Name
		}

		srcField, found := srcType.FieldByName(schemaField.Name)
		if !found {
			continue
		}
		srcFieldVal := srcVal.FieldByName(srcField.Name)
		expectedType := schemaField.Type

		srcTypeVal := srcFieldVal.Type()

		if conv, ok := getConverter(srcTypeVal, expectedType); ok {
			converted, err := safeConvert(conv, srcFieldVal.Interface())
			if err != nil {
				return nil, fmt.Errorf("cant convert field %s: %v", schemaField.Name, err)
			}
			convVal := reflect.ValueOf(converted)
			if !convVal.Type().AssignableTo(expectedType) {
				continue
			}
			result[key] = converted
			continue
		}

		if srcTypeVal.AssignableTo(expectedType) {
			result[key] = srcFieldVal.Interface()
			continue
		}

		if srcTypeVal.ConvertibleTo(expectedType) {
			convVal := srcFieldVal.Convert(expectedType)
			result[key] = convVal.Interface()
			continue
		}
	}
	return result, nil
}

func ExtractDBFields(dbModel interface{}) []string {
	val := reflect.ValueOf(dbModel)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return []string{}
	}
	typ := val.Type()
	var fields []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = field.Name
		}
		fields = append(fields, dbTag)
	}
	return fields
}

func DBModelToMap(dbModel interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	val := reflect.ValueOf(dbModel)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("dbModel must be a struct or pointer to a struct")
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = field.Name
		}

		fv := val.Field(i)
		if !fv.CanInterface() {
			continue
		}

		result[dbTag] = fv.Interface()
	}

	return result, nil
}
