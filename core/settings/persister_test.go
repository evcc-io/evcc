package settings

import (
	"reflect"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	var conf interface{}

	expect := "val"
	res := deepMerge(conf, nil, expect)
	t.Logf("%+v", res)

	if res != expect {
		t.Errorf("expected %+v, got %+v", expect, res)
	}
}

func TestDeepMerge2(t *testing.T) {
	var conf interface{}
	var res interface{} = conf

	res = deepMerge(res, []string{"foo"}, "bar")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": "bar"}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}

	res = deepMerge(res, []string{"foo"}, "baz")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": "baz"}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}

	res = deepMerge(res, []string{"baz"}, "bar")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": "baz", "baz": "bar"}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}
}

func TestDeepMergeSlice(t *testing.T) {
	var conf interface{}
	var res interface{} = conf

	res = deepMerge(res, []string{"foo", "0"}, "foo")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": []interface{}{"foo"}}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}

	res = deepMerge(res, []string{"foo", "2"}, "bar")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": []interface{}{"foo", nil, "bar"}}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}

	res = deepMerge(res, []string{"foo", "0"}, "baz")
	t.Logf("%+v", res)
	if expect := map[string]interface{}{"foo": []interface{}{"baz", nil, "bar"}}; !reflect.DeepEqual(expect, res) {
		t.Errorf("expected %+v, got %+v", expect, res)
	}

}
