package configure

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/thoas/go-funk"
)

// setDefaultTexts sets default language specific texts
func (c *CmdConfigure) setDefaultTexts() {
	c.errItemNotPresent = errors.New(c.localizedString("Error_ItemNotPresent", nil))
	c.errDeviceNotValid = errors.New(c.localizedString("Error_DeviceNotValid", nil))

	c.updateDeviceCategoryTexts(DeviceCategoryCharger, "Category_ChargerTitle", "Category_ChargerArticle", "Category_ChargerAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryGuidedSetup, "Category_SystemTitle", "Category_SystemArticle", "Category_SystemAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryGridMeter, "Category_GridMeterTitle", "Category_GridMeterArticle", "Category_GridMeterAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryPVMeter, "Category_PVMeterTitle", "Category_PVMeterArticle", "Category_PVMeterAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryBatteryMeter, "Category_BatteryMeter", "Category_BatteryMeterArticle", "Category_BatteryMeterAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryChargeMeter, "Category_ChargeMeterTitle", "Category_ChargeMeterArticle", "Category_ChargeMeterAdditional")
	c.updateDeviceCategoryTexts(DeviceCategoryVehicle, "Category_VehicleTitle", "Category_VehicleArticle", "Category_VehicleAdditional")
}

// updateDeviceCategoryTexts updates the texts for a device category
func (c *CmdConfigure) updateDeviceCategoryTexts(category DeviceCategory, title, article, additional string) {
	data := DeviceCategories[category]
	data.title = c.localizedString(title, nil)
	data.article = c.localizedString(article, nil)
	data.additional = c.localizedString(additional, nil)
	DeviceCategories[category] = data
}

// localizedString is a helper for getting a localized string
func (c *CmdConfigure) localizedString(key string, templateData localizeMap) string {
	return c.localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
	})
}

// userFriendlyTexts sets user friendly texts for a set of known param names and
// returns them as a new set as we don't want to overwrite the template values
func (c *CmdConfigure) userFriendlyTexts(param templates.Param) templates.Param {
	result := param

	if result.ValueType == "" || (result.ValueType != "" && !funk.ContainsString(templates.ParamValueTypes, result.ValueType)) {
		result.ValueType = templates.ParamValueTypeString
	}

	type replacements struct {
		Name      string
		Help      string
		Example   string
		ValueType string
	}
	resultNameMap := map[string]replacements{
		"title":        {Name: "UserFriendly_Title_Name", Help: "UserFriendly_Title_Help"},
		"device":       {Name: "UserFriendly_Device_Name", Help: "UserFriendly_Device_Help", Example: "/dev/ttyUSB0"},
		"baudrate":     {Name: "UserFriendly_Baudrate_Name", Help: "UserFriendly_Baudrate_Help", Example: "9600"},
		"comset":       {Name: "UserFriendly_ComSet_Name", Help: "UserFriendly_ComSet_Help", Example: "8N1"},
		"host":         {Name: "UserFriendly_Host_Name"},
		"port":         {Name: "UserFriendly_Port_Name", ValueType: templates.ParamValueTypeNumber},
		"user":         {Name: "UserFriendly_User_Name"},
		"password":     {Name: "UserFriendly_Password_Name"},
		"capacity":     {Name: "UserFriendly_Capacity_Name", Example: "41.5", ValueType: templates.ParamValueTypeFloat},
		"vin":          {Name: "UserFriendly_Vin_Name", Help: "UserFriendly_Vin_Help", Example: "W..."},
		"cache":        {Name: "UserFriendly_Cache_Name", Help: "UserFriendly_Cache_Help", Example: "5m"},
		"mode":         {Name: "ChargeMode_Question", ValueType: templates.ParamValueTypeChargeModes},
		"minsoc":       {Name: "UserFriendly_MinSoC_Name", Help: "UserFriendly_MinSoC_Help", Example: "25", ValueType: templates.ParamValueTypeNumber},
		"targetsoc":    {Name: "UserFriendly_TargetSoC_Name", Help: "UserFriendly_TargetSoC_Help", Example: "80", ValueType: templates.ParamValueTypeNumber},
		"mincurrent":   {Name: "UserFriendly_MinCurrent_Name", Help: "UserFriendly_MinCurrent_Help", Example: "6", ValueType: templates.ParamValueTypeNumber},
		"maxcurrent":   {Name: "UserFriendly_MaxCurrent_Name", Help: "UserFriendly_MaxCurrent_Help", Example: "16", ValueType: templates.ParamValueTypeNumber},
		"identifiers":  {Name: "UserFriendly_Identifier_Name", Help: "UserFriendly_Identifier_Help"},
		"standbypower": {Name: "UserFriendly_StandByPower_Name", Help: "UserFriendly_StandByPower_Help", ValueType: templates.ParamValueTypeNumber},
	}

	if resultMapItem := resultNameMap[strings.ToLower(result.Name)]; resultMapItem != (replacements{}) {
		// always overwrite if defined
		if resultMapItem.Name != "" {
			result.Name = c.localizedString(resultMapItem.Name, nil)
		}
		if resultMapItem.ValueType != "" {
			result.ValueType = resultMapItem.ValueType
		}
		// only set if empty
		if result.Help.String(c.lang) == "" && resultMapItem.Help != "" {
			result.Help.SetString(c.lang, c.localizedString(resultMapItem.Help, nil))
		}
		if result.Example == "" && resultMapItem.Example != "" {
			result.Example = resultMapItem.Example
		}
	}

	return result
}
