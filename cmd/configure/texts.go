package configure

import (
	"errors"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// setDefaultTexts sets default language specific texts
func (c *CmdConfigure) setDefaultTexts() {
	c.errItemNotPresent = errors.New(c.localizedString("Error_ItemNotPresent"))
	c.errDeviceNotValid = errors.New(c.localizedString("Error_DeviceNotValid"))

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
	data.title = c.localizedString(title)
	data.article = c.localizedString(article)
	data.additional = c.localizedString(additional)
	DeviceCategories[category] = data
}

// localizedString is a helper for getting a localized string
func (c *CmdConfigure) localizedString(key string, data ...localizeMap) string {
	var templateData localizeMap
	if len(data) == 1 {
		templateData = data[0]
	}
	return c.localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
	})
}
