package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestSettingsSetJsonHandler(t *testing.T) {
	type TestStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	key := "testKey"
	valueChan := make(chan util.Param, 1)

	newStruc := func() any {
		return &TestStruct{}
	}

	handler := settingsSetJsonHandler(key, valueChan, newStruc)

	t.Run("valid JSON input", func(t *testing.T) {
		input := TestStruct{
			Field1: "value1",
			Field2: 42,
		}
		body, _ := json.Marshal(input)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		select {
		case param := <-valueChan:
			assert.Equal(t, key, param.Key)
			assert.Equal(t, &input, param.Val)
		default:
			t.Error("expected value to be sent to valueChan")
		}
	})

	t.Run("invalid JSON input", func(t *testing.T) {
		body := []byte(`{"field1": "value1", "field2": "invalid"}`)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unknown fields in JSON input", func(t *testing.T) {
		body := []byte(`{"field1": "value1", "field2": 42, "unknownField": "value"}`)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestMergeSettingsOld(t *testing.T) {

	t.Run("old is nil", func(t *testing.T) {
		struc := &TestStruct{
			Field1: "newValue1",
			Field2: 42,
		}

		mergeSettingsOld(struc, nil)

		assert.Equal(t, "newValue1", struc.Field1)
		assert.Equal(t, 42, struc.Field2)
	})

	t.Run("old does not implement Redactor", func(t *testing.T) {
		struc := &TestStruct{
			Field1: "newValue1",
			Field2: 42,
		}
		old := &TestStruct{
			Field1: "oldValue1",
			Field2: 24,
		}

		mergeSettingsOld(struc, old)

		assert.Equal(t, "newValue1", struc.Field1)
		assert.Equal(t, 42, struc.Field2)
	})

	t.Run("redacted fields are replaced with old values", func(t *testing.T) {
		old := &RedactedStruct{
			Field1: "oldValue1",
			Field2: 24,
		}

		struc := &RedactedStruct{
			Field1: "redacted",
			Field2: 42,
		}

		mergeSettingsOld(struc, old)

		assert.Equal(t, "oldValue1", struc.Field1)
		assert.Equal(t, 42, struc.Field2)
	})

	t.Run("no redacted fields match", func(t *testing.T) {
		old := &RedactedStruct{
			Field1: "oldValue1",
			Field2: 24,
		}

		struc := &RedactedStruct{
			Field1: "newValue1",
			Field2: 42,
		}

		mergeSettingsOld(struc, old)

		assert.Equal(t, "newValue1", struc.Field1)
		assert.Equal(t, 42, struc.Field2)
	})
}

func TestSettingsDeleteJsonHandler(t *testing.T) {
	key := "testKey"
	valueChan := make(chan util.Param, 1)
	struc := &TestStruct{
		Field1: "value1",
		Field2: 42,
	}

	handler := settingsDeleteJsonHandler(key, valueChan, struc)

	t.Run("successful delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		select {
		case param := <-valueChan:

			assert.Equal(t, key, param.Key)
			assert.Equal(t, struc, param.Val)
		default:
			t.Error("expected value to be sent to valueChan")
		}
	})
}

type TestStruct struct {
	Field1 string
	Field2 int
}

type RedactedStruct struct {
	Field1 string
	Field2 int
}

func (t *RedactedStruct) Redacted() any {
	return struct {
		Field1 string
		Field2 int
	}{
		Field1: "redacted",
		Field2: t.Field2,
	}
}
