package qparams

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Map represents qparams map[string]string type
	Map map[string]string

	// Slice represents qparams []string type
	Slice []string
)

// Slice returns string slice of qparams Slice
func (s *Slice) Slice() []string {
	return []string(*s)
}

// ToIntSlice will attempt to convert the string slice to int slice
// Will return error if it is unable to convert any member, and a
// partial slice without errornous members
func (s *Slice) ToIntSlice() ([]int, error) {
	var err error

	newSlice := []int{}

	for _, v := range s.Slice() {
		i, e := strconv.Atoi(v)
		if e != nil {
			err = fmt.Errorf("Could not convert member %s to int", v)
			continue
		}

		newSlice = append(newSlice, i)
	}

	return newSlice, err
}

// ToFloatSlice will attempt to convert the string slice to float64 slice
// Will return error if it is unable to convert any member, and a
// partial slice without errornous members
func (s *Slice) ToFloatSlice() ([]float64, error) {
	var err error

	newSlice := []float64{}

	for _, v := range s.Slice() {
		f, e := strconv.ParseFloat(v, 64)
		if e != nil {
			err = fmt.Errorf("Could not convert member %s to float", v)
			continue
		}

		newSlice = append(newSlice, f)
	}

	return newSlice, err
}

// ToIntAtIndex will attempt to convert i-th member of slice to integer
// Will return conversion error on failure and an int zero value
/*
func (s *Slice) ToIntAtIndex(i int) (int, error) {
	return 0, nil
}

// ToFloatAtIndex will attempt to convert i-th member of slice to float64
// Will return conversion error on failure and an int zero value
func (s *Slice) ToFloatAtIndex(i int) (float64, error) {
	return 0.0, nil
}
*/

// TODO - Add tofloat etc... and do the same for Map

// ErrWrongDestType is used when the provided dest is not struct pointer
var ErrWrongDestType = errors.New("Dest must be a struct pointer")

var emptyVal interface{}
var emptyInterface = reflect.ValueOf(&emptyVal).Type().Elem().Kind()

// TypeConvErrors contain errors generated upon conversion to int or float64
type TypeConvErrors []string

func (e TypeConvErrors) Error() string {
	str := ""

	for _, e := range e {
		str += fmt.Sprintf("%s\n", e)
	}

	return str
}

var separator = ","
var mapOpsTagSeparator = ","

// Parse will try to parse query params from http.Request to
// provided struct, and will return error on filure
func Parse(dest interface{}, r *http.Request) error {
	var errs = TypeConvErrors{}

	t := reflect.TypeOf(dest)
	v := reflect.ValueOf(dest)
	queryValues := r.URL.Query()

	if t.Kind() != reflect.Ptr ||
		t.Elem().Kind() != reflect.Struct {
		return ErrWrongDestType
	}

	// TODO: - Cache struct meta data

	for i := 0; i < v.Elem().NumField(); i++ {
		fieldT := t.Elem().Field(i)
		fieldV := v.Elem().Field(i)

		vv := fieldV
		// fmt.Printf("vv Kind(): %#v: isSlice: %#v: isMap: %#v\n", vv.Kind(), reflect.Slice == vv.Kind(), reflect.Map == vv.Kind())
		// fmt.Printf("vv Type(): %#v\n", vv.Type().Name())

		// if reflect.Slice == vv.Kind() {
		// 	infoValSlice(vv)
		// }
		// if reflect.Map == vv.Kind() {
		// 	infoValMap(vv)
		// }

		fieldName := strings.ToLower(fieldT.Name)
		// fmt.Println("fieldT.Name:", fieldName)

		if tagFieldName := getTag("name", fieldT); tagFieldName != "" {
			fieldName = tagFieldName
		}

		sep := getSeparator(fieldT)

		for key, val := range queryValues {
			key = strings.ToLower(key)
			// fmt.Println("key:", key)
			if fieldName != key {
				continue
			}
			r := []string{}
			// fmt.Printf("fieldT.Type.Name():__%#v\n", fieldT.Type.Name())
			switch vv.Kind() {
			case reflect.Slice, reflect.Map:
				// fmt.Printf("trim []string: %#v\n", val)
				for _, n := range val {
					r = append(r, strings.Trim(n, ","+sep))
				}
			default:
				r = val
			}
			switch vv.Kind() {
			case reflect.Map:
				t := vv.Type()
				// allocate a new map, if v is nil. see: m2, m3, m4.
				if vv.IsNil() {
					vv.Set(reflect.MakeMap(t))
				}
				if t.Key().Name() != "string" || t.Elem().Kind() != reflect.String {
					continue
				}
				queryValues[key] = []string{strings.Join(r, ",")}
			case reflect.Slice:
				t := vv.Type()
				// allocate a new map, if v is nil. see: m2, m3, m4.
				if vv.IsNil() {
					vv.Set(reflect.MakeSlice(t, 0, 0))
				}
				if t.Elem().String() != "string" {
					continue
				}
				// fmt.Printf("join []string: %#v\n", r)
				queryValues[key] = []string{strings.Join(r, sep)}
			default:
				queryValues[key] = r
			}
		}

		queryValue := queryValues.Get(fieldName)

		if queryValue == "" {
			// TODO - Set default value here
			// fmt.Println("-------------------------")
			continue
		}

		switch vv.Kind() {
		case reflect.Map:
			parseMap(fieldT, fieldV, queryValue)
		case reflect.Slice:
			// fmt.Printf("fill []string: %#v\n", queryValue)
			parseSlice(fieldT, fieldV, queryValue)
		}

		switch fieldV.Kind() {
		case reflect.Int, reflect.Int32:
			err := parseInt(fieldT, fieldV, queryValue)
			if err != nil {
				errs = append(errs, err.Error())
			}
		case reflect.Int64:
			err := parseInt64(fieldT, fieldV, queryValue)
			if err != nil {
				errs = append(errs, err.Error())
			}
		case reflect.Float64:
			err := parseFloat(fieldT, fieldV, queryValue, 64)
			if err != nil {
				errs = append(errs, err.Error())
			}
		case reflect.Float32:
			err := parseFloat(fieldT, fieldV, queryValue, 32)
			if err != nil {
				errs = append(errs, err.Error())
			}
		case reflect.String:
			parseString(fieldT, fieldV, queryValue)
		}

		// fmt.Println("-------------------------")
	}

	if errs != nil && len(errs) > 0 {
		return errs
	}

	return nil
}

func getTag(tag string, sField reflect.StructField) string {
	tags := sField.Tag.Get("qparams")

	if tags == "" {
		return tags
	}

	tagSlice := strings.Split(tags, " ")

	for _, t := range tagSlice {
		subSlice := strings.Split(t, ":")

		if subSlice != nil &&
			len(subSlice) == 2 &&
			subSlice[0] == tag {
			return subSlice[1]
		}
	}

	return ""
}

func getSeparator(sField reflect.StructField) string {
	sep := separator

	if s := getTag("sep", sField); s != "" {
		sep = s
	}

	return sep
}

func getOperators(sField reflect.StructField) []string {
	operators := []string{}

	if ops := getTag("ops", sField); ops != "" {
		operators = strings.Split(ops, mapOpsTagSeparator)
	}

	return operators
}

func parseMap(sField reflect.StructField, fieldV reflect.Value, queryValue string) {
	sep := getSeparator(sField)

	operators := getOperators(sField)
	// TODO: - Throw error if no operators provided

	// TODO - handle error
	parsedMap := walk(queryValue, sep, operators)

	fieldV.Set(reflect.ValueOf(parsedMap))
}

func parseSlice(sField reflect.StructField, fieldV reflect.Value, queryValue string) {
	sep := getSeparator(sField)

	slice := strings.Split(queryValue, sep)

	newSlice := []string{}

	for _, val := range slice {
		v := strings.ToLower(val)
		if v != "" {
			newSlice = append(newSlice, v)
		}
	}

	fieldV.Set(reflect.ValueOf(newSlice))
}

func parseInt(sField reflect.StructField, fieldV reflect.Value, queryValue string) error {
	i, err := strconv.Atoi(queryValue)
	if err != nil {
		return fmt.Errorf("Field %s does not contain a valid integer (%s)", sField.Name, queryValue)
	}

	fieldV.Set(reflect.ValueOf(i))

	return nil
}

func parseInt64(sField reflect.StructField, fieldV reflect.Value, queryValue string) error {
	i64, err := strconv.ParseInt(queryValue, 10, 0)
	if err != nil {
		return fmt.Errorf("Field %s does not contain a valid integer (%s)", sField.Name, queryValue)
	}

	fieldV.Set(reflect.ValueOf(i64))

	return nil
}

func parseFloat(sField reflect.StructField, fieldV reflect.Value, queryValue string, bitSize int) error {
	f, err := strconv.ParseFloat(queryValue, bitSize)
	if err != nil {
		return fmt.Errorf("Field %s does not contain a valid float (%s)", sField.Name, queryValue)
	}

	fieldV.Set(reflect.ValueOf(f))

	return nil
}

func parseString(sField reflect.StructField, fieldV reflect.Value, queryValue string) {
	fieldV.Set(reflect.ValueOf(queryValue))
}

// func infoValMap(dst reflect.Value) {
// 	if reflect.Map != dst.Kind() {
// 		return
// 	}
// 	t := dst.Type()
// 	// allocate a new map, if v is nil. see: m2, m3, m4.
// 	if dst.IsNil() {
// 		dst.Set(reflect.MakeMap(t))
// 	}

// 	if t.Key().Name() != "string" || (t.Elem().Kind() != reflect.String && t.Elem().Kind() != emptyInterface) {
// 		return
// 	}
// 	fmt.Printf("Map vv.Kind(): %#v\n", dst.Kind().String())
// 	fmt.Printf("Index Type: %#v\n", t.Key().Name())
// 	fmt.Printf("Value Type: %#v\n", t.Elem().String())
// }
// func infoValSlice(dst reflect.Value) {
// 	if reflect.Slice != dst.Kind() {
// 		return
// 	}
// 	t := dst.Type()
// 	// allocate a new map, if v is nil. see: m2, m3, m4.
// 	if dst.IsNil() {
// 		dst.Set(reflect.MakeSlice(t, 0, 0))
// 	}
// 	if t.Elem().String() != "string" {
// 		return
// 	}
// 	fmt.Printf("Slice vv.Kind(): %#v\n", dst.Kind().String())
// 	fmt.Printf("Value Type: %#v\n", t.Elem().String())
// }
