package config

import (
	"fmt"

	"github.com/evcc-io/evcc/util/templates"
	"gorm.io/gorm"
)

type Config struct {
	ID      int `gorm:"primarykey"`
	Class   templates.Class
	Type    string
	Details []ConfigDetail `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type ConfigDetail struct {
	ConfigID int    `gorm:"index:idx_unique"`
	Key      string `gorm:"index:idx_unique"`
	Value    string
}

// Named converts device details to named config
func (d *Config) Named() Named {
	res := Named{
		Name:  NameForID(d.ID),
		Type:  d.Type,
		Other: d.detailsAsMap(),
	}
	return res
}

// Typed converts device details to typed config
func (d *Config) Typed() Typed {
	res := Typed{
		Type:  d.Type,
		Other: d.detailsAsMap(),
	}
	return res
}

// detailsAsMap converts device details to map
func (d *Config) detailsAsMap() map[string]any {
	res := make(map[string]any, len(d.Details))
	for _, detail := range d.Details {
		res[detail.Key] = detail.Value
	}
	return res
}

// detailsFromMap converts map to device details
func detailsFromMap(config map[string]any) []ConfigDetail {
	res := make([]ConfigDetail, 0, len(config))
	for k, v := range config {
		res = append(res, ConfigDetail{Key: k, Value: fmt.Sprintf("%v", v)})
	}
	return res
}

// Update updates a config's details to the database
func (d *Config) Update(conf map[string]any) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var config Config
		if err := tx.Where(Config{Class: d.Class, ID: d.ID}).First(&config).Error; err != nil {
			return err
		}

		if err := tx.Delete(new(ConfigDetail), ConfigDetail{ConfigID: d.ID}).Error; err != nil {
			return err
		}

		d.Details = detailsFromMap(conf)

		return tx.Save(&d).Error
	})
}

// Delete deletes a config from the database
func (d *Config) Delete() error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(new(ConfigDetail), ConfigDetail{ConfigID: d.ID}).Error; err != nil {
			return err
		}

		return tx.Delete(Config{ID: d.ID}).Error
	})
}

var db *gorm.DB

func Init(instance *gorm.DB) error {
	db = instance

	for old, new := range map[string]string{
		"devices":        "configs",
		"device_details": "config_details",
	} {
		if m := db.Migrator(); m.HasTable(old) {
			if err := m.RenameTable(old, new); err != nil {
				return err
			}
		}
	}

	err := db.AutoMigrate(new(Config), new(ConfigDetail))

	if err == nil && db.Migrator().HasConstraint(new(ConfigDetail), "fk_devices_details") {
		err = db.Migrator().DropConstraint(new(ConfigDetail), "fk_devices_details")
	}
	if err == nil && db.Migrator().HasColumn(new(ConfigDetail), "device_id") {
		err = db.Migrator().DropColumn(new(ConfigDetail), "device_id")
	}

	return err
}

// NameForID returns a unique config name for the given id
func NameForID(id int) string {
	return fmt.Sprintf("db:%d", id)
}

// ConfigurationsByClass returns devices by class from the database
func ConfigurationsByClass(class templates.Class) ([]Config, error) {
	var devices []Config
	tx := db.Where(&Config{Class: class}).Preload("Details").Order("id").Find(&devices)

	// remove devices without details
	res := make([]Config, 0, len(devices))
	for _, dev := range devices {
		if len(dev.Details) > 0 {
			res = append(res, dev)
		}
	}

	return res, tx.Error
}

// ConfigByID returns device by id from the database
func ConfigByID(id int) (Config, error) {
	var config Config
	tx := db.Where(&Config{ID: id}).Preload("Details").First(&config)
	return config, tx.Error
}

// AddConfig adds a new config to the database
func AddConfig(class templates.Class, typ string, conf map[string]any) (Config, error) {
	config := Config{
		Class:   class,
		Type:    typ,
		Details: detailsFromMap(conf),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&config).Error
	})

	return config, err
}
