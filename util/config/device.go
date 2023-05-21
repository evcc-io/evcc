package config

import (
	"fmt"

	"gorm.io/gorm"
)

type Device struct {
	ID      int `gorm:"primarykey"`
	Class   Class
	Type    string
	Details []DeviceDetail `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type DeviceDetail struct {
	DeviceID int    `gorm:"primarykey"`
	Key      string `gorm:"primarykey"`
	Value    string
}

// DetailsAsMap converts device details to map
func (d *Device) DetailsAsMap() map[string]any {
	res := make(map[string]any)
	for _, detail := range d.Details {
		res[detail.Key] = detail.Value
	}
	return res
}

// mapAsDetails converts device details to map
func (d *Device) mapAsDetails(config map[string]any) []DeviceDetail {
	res := make([]DeviceDetail, 0, len(config))
	for k, v := range config {
		res = append(res, DeviceDetail{DeviceID: d.ID, Key: k, Value: fmt.Sprintf("%v", v)})
	}
	return res
}

var db *gorm.DB

func Init(instance *gorm.DB) error {
	db = instance
	return db.AutoMigrate(new(Device), new(DeviceDetail))
}

// NameForID returns a unique config name for the given id
func NameForID(id int) string {
	return fmt.Sprintf("db:%d", id)
}

// Devices returns devices by class from the database
func Devices(class Class) ([]Device, error) {
	var devices []Device
	tx := db.Where(&Device{Class: class}).Preload("Details").Order("id").Find(&devices)

	// remove devices without details
	for i := 0; i < len(devices); {
		if len(devices[i].Details) > 0 {
			i++
			continue
		}

		// delete device
		copy(devices[i:], devices[i+1:])
		devices = devices[: len(devices)-1 : len(devices)-1]
	}

	return devices, tx.Error
}

// DeviceByID returns device by id from the database
func DeviceByID(id int) (Device, error) {
	var device Device
	tx := db.Where(&Device{ID: id}).Preload("Details").First(&device)
	return device, tx.Error
}

// AddDevice adds a new device to the database
func AddDevice(class Class, typ string, config map[string]any) (int, error) {
	device := Device{Class: class, Type: typ}
	if tx := db.Create(&device); tx.Error != nil {
		return 0, tx.Error
	}

	details := device.mapAsDetails(config)
	tx := db.Create(&details)

	return device.ID, tx.Error
}

// UpdateDevice updates a device's details to the database
func UpdateDevice(class Class, id int, config map[string]any) (int64, error) {
	var device Device
	if tx := db.Where(Device{Class: class, ID: id}).First(&device); tx.Error != nil {
		return 0, tx.Error
	}

	if tx := db.Where(DeviceDetail{DeviceID: id}); tx.Error != nil {
		return 0, tx.Error
	} else if tx.RowsAffected > 0 {
		if tx := db.Delete(DeviceDetail{DeviceID: id}); tx.Error != nil {
			return 0, tx.Error
		}
	}

	details := device.mapAsDetails(config)
	tx := db.Save(&details)

	return tx.RowsAffected, tx.Error
}

// DeleteDevice deletes a device from the database
func DeleteDevice(class Class, id int) (int64, error) {
	if tx := db.Where(DeviceDetail{DeviceID: id}); tx.Error != nil {
		return 0, tx.Error
	} else if tx.RowsAffected > 0 {
		if tx := db.Delete(DeviceDetail{DeviceID: id}); tx.Error != nil {
			return 0, tx.Error
		}
	}

	tx := db.Delete(Device{ID: id})
	return tx.RowsAffected, tx.Error
}
