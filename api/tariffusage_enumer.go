// Code generated by "enumer -type TariffUsage -trimprefix TariffUsage -transform=lower"; DO NOT EDIT.

package api

import (
	"fmt"
	"strings"
)

const _TariffUsageName = "co2feedingridplannersolar"

var _TariffUsageIndex = [...]uint8{0, 3, 9, 13, 20, 25}

const _TariffUsageLowerName = "co2feedingridplannersolar"

func (i TariffUsage) String() string {
	i -= 1
	if i < 0 || i >= TariffUsage(len(_TariffUsageIndex)-1) {
		return fmt.Sprintf("TariffUsage(%d)", i+1)
	}
	return _TariffUsageName[_TariffUsageIndex[i]:_TariffUsageIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _TariffUsageNoOp() {
	var x [1]struct{}
	_ = x[TariffUsageCo2-(1)]
	_ = x[TariffUsageFeedIn-(2)]
	_ = x[TariffUsageGrid-(3)]
	_ = x[TariffUsagePlanner-(4)]
	_ = x[TariffUsageSolar-(5)]
}

var _TariffUsageValues = []TariffUsage{TariffUsageCo2, TariffUsageFeedIn, TariffUsageGrid, TariffUsagePlanner, TariffUsageSolar}

var _TariffUsageNameToValueMap = map[string]TariffUsage{
	_TariffUsageName[0:3]:        TariffUsageCo2,
	_TariffUsageLowerName[0:3]:   TariffUsageCo2,
	_TariffUsageName[3:9]:        TariffUsageFeedIn,
	_TariffUsageLowerName[3:9]:   TariffUsageFeedIn,
	_TariffUsageName[9:13]:       TariffUsageGrid,
	_TariffUsageLowerName[9:13]:  TariffUsageGrid,
	_TariffUsageName[13:20]:      TariffUsagePlanner,
	_TariffUsageLowerName[13:20]: TariffUsagePlanner,
	_TariffUsageName[20:25]:      TariffUsageSolar,
	_TariffUsageLowerName[20:25]: TariffUsageSolar,
}

var _TariffUsageNames = []string{
	_TariffUsageName[0:3],
	_TariffUsageName[3:9],
	_TariffUsageName[9:13],
	_TariffUsageName[13:20],
	_TariffUsageName[20:25],
}

// TariffUsageString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func TariffUsageString(s string) (TariffUsage, error) {
	if val, ok := _TariffUsageNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _TariffUsageNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to TariffUsage values", s)
}

// TariffUsageValues returns all values of the enum
func TariffUsageValues() []TariffUsage {
	return _TariffUsageValues
}

// TariffUsageStrings returns a slice of all String values of the enum
func TariffUsageStrings() []string {
	strs := make([]string, len(_TariffUsageNames))
	copy(strs, _TariffUsageNames)
	return strs
}

// IsATariffUsage returns "true" if the value is listed in the enum definition. "false" otherwise
func (i TariffUsage) IsATariffUsage() bool {
	for _, v := range _TariffUsageValues {
		if i == v {
			return true
		}
	}
	return false
}
