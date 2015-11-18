package env

import (
	"reflect"
	"fmt"
	"os"
	"strings"
	"strconv"
)

type InvalidIndirectError struct {
	Type reflect.Type
}

func (err *InvalidIndirectError) Error() string {
	if k := err.Type.Kind(); k != reflect.Struct && k != reflect.Map {
		return fmt.Sprintf("env: cannot unmarshal to the type %s", k)
	}
	return "env: destination map is not in the format map[string]string"
}

// rv must be a non-nil pointer or a settable value
func indirect(rv reflect.Value) (reflect.Value, error) {
	for {
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			if e := rv.Elem(); e.Kind() == reflect.Ptr && !e.IsNil() {
				rv = e.Elem()
			}
		}

		if rv.Kind() == reflect.Map && rv.IsNil() {
			rv.Set(reflect.MakeMap(rv.Type()))
		}

		if rv.Kind() != reflect.Ptr {
			break
		}

		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}
	if k := rv.Kind(); k != reflect.Struct && k != reflect.Map {
		return reflect.Value{}, &InvalidIndirectError{rv.Type()}
	}
	if rv.Kind() == reflect.Map {
		if t := rv.Type(); t.Key().Kind() != reflect.String || t.Elem().Kind() != reflect.String {
			return reflect.Value{}, &InvalidIndirectError{t}
		}
	}
	return rv, nil
}

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (err *InvalidUnmarshalError) Error() string {
	if err.Type == nil {
		return "env: Unmarshal(nil)"
	}
	if err.Type.Kind() != reflect.Ptr {
		return fmt.Sprintf("env: Unmarshal(non-pointer %s)", err.Type)
	}
	return "env: Unmarshal(nil-pointer)"
}

func Unmarshal(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(rv)}
	}

	rv, err := indirect(rv)
	if err != nil {
		return err
	}
	if rv.Kind() == reflect.Struct {
		return unmarshalStruct(rv)
	}
	return unmarshalMap(rv)
}

type tagOpts struct {
	ignored   bool
	omitEmpty bool
	alias     string
}

func parseTag(tag reflect.StructTag) *tagOpts {
	opts := &tagOpts{}
	switch s := tag.Get("env"); s {
	case "":
	case "-":
		opts.ignored = true
	default:
		for k, v := range strings.Split(s, ",") {
			if k == 0 {
				opts.alias = v
				continue
			}
			if v == "omitempty" {
				opts.omitEmpty = true
			}
		}
	}
	return opts
}

type UnmarshalTypeError struct {
	Name  string
	Value string
	Type  reflect.Type
}

func (err *UnmarshalTypeError) Error() string {
	fs := "env: cannot unmarshal %s's %s into type %s"
	return fmt.Sprintf(fs, err.Name, err.Value, err.Type.String())
}

func unmarshalStruct(rv reflect.Value) error {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		frv := rv.Field(i)
		if !frv.CanSet() {
			continue
		}

		f := rt.Field(i)
		opts := parseTag(f.Tag)
		if opts.ignored {
			continue
		}
		name := map[bool]string{true: opts.alias, false: f.Name}[opts.alias != ""]
		value := os.Getenv(name)
		if value == "" {
			if !opts.omitEmpty {
				return fmt.Errorf("env: environment variable %s is not set", name)
			}
			continue
		}

		frt := frv.Type()
		switch frt.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(value, 10, frt.Bits())
			if err != nil {
				return &UnmarshalTypeError{name, value, frt}
			}
			frv.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(value, 10, frt.Bits())
			if err != nil {
				return &UnmarshalTypeError{name, value, frt}
			}
			frv.SetUint(u)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(value, frt.Bits())
			if err != nil {
				return &UnmarshalTypeError{name, value, frt}
			}
			frv.SetFloat(f)
		case reflect.String:
			frv.SetString(value)
		default:
			return &UnmarshalTypeError{name, value, frt}
		}
	}
	return nil
}

func unmarshalMap(rv reflect.Value) error {
	for _, key := range rv.MapKeys() {
		rv.SetMapIndex(key, reflect.ValueOf(os.Getenv(key.Interface().(string))))
	}
	return nil
}
