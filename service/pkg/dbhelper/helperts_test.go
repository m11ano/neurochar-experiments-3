package dbhelper

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func intToStringConverter(src interface{}) (interface{}, error) {
	val, ok := src.(int)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", src)
	}
	return fmt.Sprintf("%d", val), nil
}

func stringToIntConverter(src interface{}) (interface{}, error) {
	val, ok := src.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", src)
	}
	if val == "panic" {
		panic("converter panic example")
	}
	return 42, nil
}

func TestRegisterBidirectionalConverter(t *testing.T) {
	RegisterBidirectionalConverter(
		reflect.TypeOf(int(0)),
		reflect.TypeOf(""),
		intToStringConverter,
		stringToIntConverter,
	)

	conv, found := getConverter(reflect.TypeOf(int(0)), reflect.TypeOf(""))
	assert.True(t, found)
	res, err := safeConvert(conv, 123)
	assert.NoError(t, err)
	assert.Equal(t, "123", res)

	conv2, found2 := getConverter(reflect.TypeOf(""), reflect.TypeOf(int(0)))
	assert.True(t, found2)
	res2, err2 := safeConvert(conv2, "any")
	assert.NoError(t, err2)
	assert.EqualValues(t, 42, res2)
}

func TestSafeConvert_Panic(t *testing.T) {
	RegisterBidirectionalConverter(
		reflect.TypeOf("panic"),
		reflect.TypeOf(int(0)),
		stringToIntConverter,
		nil,
	)

	conv, ok := getConverter(reflect.TypeOf("panic"), reflect.TypeOf(int(0)))
	assert.True(t, ok)
	res, err := safeConvert(conv, "panic")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "converter panic example")
	assert.Nil(t, res)
}

type DBUser struct {
	ID         int    `db:"id"`
	FirstName  string `db:"first_name"`
	BirthDate  string `db:"birth_date"`
	UserStatus string `db:"status"`
	RawField   bool   `db:"raw_field"`
}

type DomainUser struct {
	ID         int
	FirstName  string
	BirthDate  string
	UserStatus string
	RawField   bool
}

func TestConvertDBToDomain_SimpleAssign(t *testing.T) {
	dbObj := DBUser{
		ID:         10,
		FirstName:  "Alice",
		BirthDate:  "1990-01-01",
		UserStatus: "active",
		RawField:   true,
	}
	var domainObj DomainUser

	err := ConvertDBToDomain(dbObj, &domainObj)
	assert.NoError(t, err)
	assert.Equal(t, 10, domainObj.ID)
	assert.Equal(t, "Alice", domainObj.FirstName)
	assert.Equal(t, "1990-01-01", domainObj.BirthDate)
	assert.Equal(t, "active", domainObj.UserStatus)
	assert.True(t, domainObj.RawField)
}

func TestConvertDBToDomain_FieldNameMismatch(t *testing.T) {
	type DBModel struct {
		Foo string
	}
	type DomainModel struct {
		Bar string
	}
	dbObj := DBModel{Foo: "value"}
	var domainObj DomainModel

	err := ConvertDBToDomain(dbObj, &domainObj)
	assert.NoError(t, err, "ошибки нет, библиотека просто пропустит незнакомое поле")
	assert.Empty(t, domainObj.Bar, "поле Bar остаётся пустым, т.к. DBModel.Foo != DomainModel.Bar")
}

func TestConvertDBToDomain_ConverterUsage(t *testing.T) {
	type DBModel struct {
		Count int
	}
	type DomainModel struct {
		Count string
	}
	RegisterBidirectionalConverter(
		reflect.TypeOf(int(0)),
		reflect.TypeOf(""),
		intToStringConverter,
		stringToIntConverter,
	)

	dbObj := DBModel{Count: 999}
	var domainObj DomainModel

	err := ConvertDBToDomain(dbObj, &domainObj)
	assert.NoError(t, err)
	assert.Equal(t, "999", domainObj.Count, "должен использоваться пользовательский конвертер, а не int->rune->string")
}

func TestConvertDBToDomain_NoConverterFound(t *testing.T) {
	type DBModel struct {
		Foo complex64
	}
	type DomainModel struct {
		Foo string
	}
	dbObj := DBModel{Foo: 1 + 2i}
	var domainObj DomainModel

	err := ConvertDBToDomain(dbObj, &domainObj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no converter registered for field Foo")
}

func TestConvertDBToDomain_InvalidArg(t *testing.T) {
	err := ConvertDBToDomain(struct{}{}, struct{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domainModel must be a pointer to a struct")

	var str string
	err2 := ConvertDBToDomain(str, &struct{}{})
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "dbRecord must be a struct or pointer to a struct")
}

func TestStructToDBMap_SimpleAssign(t *testing.T) {
	dbSchema := DBUser{}
	domainObj := DomainUser{
		ID:         11,
		FirstName:  "Bob",
		BirthDate:  "2000-05-05",
		UserStatus: "premium",
		RawField:   false,
	}

	dbMap, err := StructToDBMap(domainObj, dbSchema)
	assert.NoError(t, err)

	assert.EqualValues(t, 11, dbMap["id"])
	assert.EqualValues(t, "Bob", dbMap["first_name"])
	assert.EqualValues(t, "2000-05-05", dbMap["birth_date"])
	assert.EqualValues(t, "premium", dbMap["status"])
	assert.EqualValues(t, false, dbMap["raw_field"])
}

func TestStructToDBMap_ConverterUsage(t *testing.T) {
	type DBModel struct {
		Count int `db:"count"`
	}
	type DomainModel struct {
		Count string
	}

	RegisterBidirectionalConverter(
		reflect.TypeOf(int(0)),
		reflect.TypeOf(""),
		intToStringConverter,
		stringToIntConverter,
	)

	dbSchema := DBModel{}
	domainObj := DomainModel{Count: "1234"}

	dbMap, err := StructToDBMap(domainObj, dbSchema)
	assert.NoError(t, err)

	val, ok := dbMap["count"]
	assert.True(t, ok, "ключ 'count' должен существовать в dbMap")
	assert.EqualValues(t, 42, val, "string '1234' → int 42")
}

func TestStructToDBMap_NoConverter(t *testing.T) {
	type DBModel struct {
		Data int `db:"data"`
	}
	type DomainModel struct {
		Data complex64
	}
	dbSchema := DBModel{}
	domainObj := DomainModel{Data: 1 + 2i}

	dbMap, err := StructToDBMap(domainObj, dbSchema)
	assert.NoError(t, err, "Функция просто пропускает поле без конвертера")
	_, ok := dbMap["data"]
	assert.False(t, ok, "Поля 'data' не должно быть, т.к. нет подходящего конвертера")
}

func TestStructToDBMap_InvalidArg(t *testing.T) {
	var invalid int
	dbSchema := DBUser{}
	_, err := StructToDBMap(invalid, dbSchema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "src must be a struct or pointer to a struct")

	_, err2 := StructToDBMap(DBUser{}, 1234)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "dbSchema must be a struct or pointer to a struct")
}

func TestExtractDBFields(t *testing.T) {
	dbUser := DBUser{}
	fields := ExtractDBFields(dbUser)
	assert.Contains(t, fields, "id")
	assert.Contains(t, fields, "first_name")
	assert.Contains(t, fields, "birth_date")
	assert.Contains(t, fields, "status")
	assert.Contains(t, fields, "raw_field")
	assert.Len(t, fields, 5)

	var num int
	assert.Empty(t, ExtractDBFields(num))
}
