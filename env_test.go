package env

import (
	"testing"
	"reflect"
	"os"
)

func TestIndirect(t *testing.T) {
	a :=  (**map[string]string)(nil)
	rv, err := indirect(reflect.ValueOf(&a))
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := rv.Interface().(map[string]string); !ok {
		t.Fatal("indirection value is not a map")
	} else if m["foo"] = "bar"; (**a)["foo"] != "bar" {
		t.Fatal("indirection value is not a settable map")
	}
}

type A map[string]string

type B struct {
	String string  `env:"STRING"`
	Float  float32 `env:"FLOAT"`
	Int    int     `env:"INT"`
}

type C struct {
	Ignore string `env:"-"`
}

type D struct {
	OmitEmpty string `env:"OMITEMPTY,omitempty"`
}

type E struct {
	Required string
}

var unmarshalTests = []struct {
	in  map[string]string
	ptr interface{}
	out interface{}
	err bool
}{
	{
		in: map[string]string{"STRING": "foo"},
		ptr: &A{"STRING": ""},
		out: A{"STRING": "foo"},
	},
	{
		in: map[string]string{"STRING": "foo", "FLOAT": "0.1", "INT": "1"},
		ptr: new(B),
		out: B{String: "foo", Float: 0.1, Int: 1},
	},
	{
		in: map[string]string{"Ignore": "foo"},
		ptr: new(C),
		out: C{},
	},
	{
		in: map[string]string{"OMITEMPTY": "foo"},
		ptr: new(D),
		out: D{OmitEmpty: "foo"},
	},
	{
		in: map[string]string{"OMITEMPTY": ""},
		ptr: new(D),
		out: D{},
	},
	{
		in: map[string]string{},
		ptr: new(E),
		err: true,
	},
}

func TestUnmarshal(t *testing.T) {
	for i, test := range unmarshalTests {
		for k, v := range test.in {
			os.Setenv(k, v)
		}
		rv := reflect.ValueOf(test.ptr)
		if err := Unmarshal(rv.Interface()); err != nil && !test.err {
			t.Errorf("%d, error: %v", i, err)
			continue
		} else if err == nil && test.err {
			t.Errorf("%d, want error")
		} else if err != nil && test.err {
			continue
		}
		if !reflect.DeepEqual(rv.Elem().Interface(), test.out) {
			t.Errorf("%d, have: %#v, want: %#v", i, rv.Elem().Interface(), test.out)
		}
	}
}
