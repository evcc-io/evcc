package foxesscloud

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// FoxESS Cloud API Docs:
// https://www.foxesscloud.com/public/i18n/en/OpenApiDocument.html
type FoxESSCloudAPI struct {
	*request.Helper
	Logger *util.Logger
	URI    string
	Key    string
}

// NewFoxESSCloudAPI create new FoxESS Cloud API
func NewFoxESSCloudAPI(key string, helper *request.Helper, logger *util.Logger) *FoxESSCloudAPI {
	foxesscloud := &FoxESSCloudAPI{
		Helper: helper,
		Logger: logger,
		URI:    "https://www.foxesscloud.com",
		Key:    key,
	}
	return foxesscloud
}

// Wrappers
func (m *FoxESSCloudAPI) GetDeviceRealTimeData(sn string, variables []string) (*GetDeviceRealTimeData, error) {
	res, err := m.DoApiGetDeviceRealTimeData(sn, variables)
	if err != nil {
		return nil, err
	}

	parsed, err := m.ParseDatas((*res)[0].Datas)
	if err != nil {
		return nil, err
	}

	result := &GetDeviceRealTimeData{
		SN:   (*res)[0].DeviceSN,
		Time: (*res)[0].Time,
		Data: *parsed,
	}

	return result, err
}

func (m *FoxESSCloudAPI) SetDeviceMinSoc(sn string, minSoc, minSocOnGrid uint8) error {
	return m.DoSetDeviceBatteryMinSoc(sn, minSoc, minSocOnGrid)
}

// API
func (m *FoxESSCloudAPI) DoApiGetDeviceRealTimeData(sn string, variables []string) (*GetDeviceRealTimeDataResult, error) {
	path := "/op/v0/device/real/query"

	params := make(map[string]interface{})

	params["sn"] = sn

	if variables != nil {
		params["variables"] = variables
	}

	gen, err := m.DoFoxESSCloud(path, "en", http.MethodPost, params)
	if err != nil {
		return nil, err
	}

	var res GetDeviceRealTimeDataResult
	err = json.Unmarshal(*gen, &res)
	if err != nil {
		return nil, err
	}

	return &res, err
}

func (m *FoxESSCloudAPI) DoSetDeviceBatteryMinSoc(sn string, minSoc, minSocOnGrid uint8) error {
	path := "/op/v0/device/battery/soc/set"

	params := make(map[string]interface{})

	params["sn"] = sn
	params["minSoc"] = minSoc
	params["minSocOnGrid"] = minSocOnGrid

	if _, err := m.DoFoxESSCloud(path, "en", http.MethodPost, params); err != nil {
		return err
	} else {
		return nil
	}
}

// Helpers
func (m *FoxESSCloudAPI) DoFoxESSCloud(path, lang, method string, params map[string]interface{}) (*json.RawMessage, error) {
	uri := m.URI + path
	headers := map[string]string{}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := path + "\\r\\n" + m.Key + "\\r\\n" + timestamp

	hash := md5.New()
	hash.Write([]byte(signature))

	headers["Token"] = m.Key
	headers["Timestamp"] = timestamp
	headers["Signature"] = hex.EncodeToString(hash.Sum([]byte(nil)))
	headers["Lang"] = lang
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"
	headers["Content-Type"] = "application/json"

	var req *http.Request
	var err error

	switch method {
	case http.MethodGet:
		{
			req, err = request.New(http.MethodGet, uri, nil, headers)
			if err != nil {
				return nil, err
			}
			q := req.URL.Query()
			for k, v := range params {
				q.Add(k, fmt.Sprintf("%v", v))
			}
			req.URL.RawQuery = q.Encode()
		}
	case http.MethodPost:
		{
			jsonData, err := json.Marshal(params)
			if err != nil {
				return nil, err
			}
			req, err = request.New(http.MethodPost, uri, bytes.NewBuffer(jsonData), headers)
			if err != nil {
				return nil, err
			}
		}
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	var gen GenericResponse
	err = m.DoJSON(req, &gen)
	if err != nil {
		return nil, err
	}

	if gen.ErrNo != 0 {
		return nil, fmt.Errorf("[%d] - %s", gen.ErrNo, gen.Msg)
	}

	return &gen.Result, nil
}

func (m *FoxESSCloudAPI) ParseDatas(datas []Data) (*Variables, error) {
	parsed := &Variables{}
	val := reflect.ValueOf(parsed).Elem()

	for _, data := range datas {
		fieldKey := strings.ToUpper(string(data.Variable[0])) + data.Variable[1:]
		fieldVal := data.Value

		field := val.FieldByName(fieldKey)

		if field.IsValid() && field.CanSet() {
			if field.Kind() == reflect.Ptr && field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}

			switch field.Interface().(type) {
			case *float64:
				var v float64
				if err := json.Unmarshal(fieldVal, &v); err == nil {
					field.Set(reflect.ValueOf(&v))
				} else {
					return nil, fmt.Errorf("invalid conversion for %s", fieldKey)
				}
			case *int:
				var v float64
				if err := json.Unmarshal(fieldVal, &v); err == nil {
					i := int(v)
					field.Set(reflect.ValueOf(&i))
				} else {
					return nil, fmt.Errorf("invalid conversion for %s", fieldKey)
				}
			case string:
				var v string
				if err := json.Unmarshal(fieldVal, &v); err == nil {
					field.Set(reflect.ValueOf(v))
				} else {
					return nil, fmt.Errorf("invalid conversion for %s", fieldKey)
				}
			// Add other type cases as needed
			default:
				return nil, fmt.Errorf("unsupported field type for %s: %T", fieldKey, field.Interface())
			}
		} else {
			return nil, fmt.Errorf("no such field %s in variables or field cannot be set", fieldKey)
		}
	}

	return parsed, nil
}