package wattpilot

import (
	"strconv"
)

type PostFunction func(interface{}) (string, error)

var postProcess = map[string]struct {
	key string
	f   PostFunction
}{
	"voltage1": {"nrg", voltage1Process},
	"voltage2": {"nrg", voltage2Process},
	"voltage3": {"nrg", voltage3Process},
	"voltageN": {"nrg", voltageNProcess},
	"amps1":    {"nrg", amps1Process},
	"amps2":    {"nrg", amps2Process},
	"amps3":    {"nrg", amps3Process},
	"power1":   {"nrg", power1Process},
	"power2":   {"nrg", power2Process},
	"power3":   {"nrg", power3Process},
	"powerM":   {"nrg", powerNProcess},
	"power":    {"nrg", powerProcess},
}

func voltage1Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 0)), nil
}

func voltage2Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 1)), nil
}

func voltage3Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 2)), nil
}

func voltageNProcess(data interface{}) (string, error) {
	return float2String(voltageData(data, 3)), nil
}

func amps1Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 4)), nil
}

func amps2Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 5)), nil
}

func amps3Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 6)), nil
}

func power1Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 7)), nil
}

func power2Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 8)), nil
}

func power3Process(data interface{}) (string, error) {
	return float2String(voltageData(data, 9)), nil
}
func powerNProcess(data interface{}) (string, error) {
	return float2String(voltageData(data, 10)), nil
}

func powerProcess(data interface{}) (string, error) {
	return float2String(voltageData(data, 11)), nil
}

func voltageData(data interface{}, idx int) float64 {
	vars := data.([]interface{})
	v := vars[idx].(float64)
	return v
}

func float2String(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
