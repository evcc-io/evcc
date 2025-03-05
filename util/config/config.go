package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util/templates"
	"gorm.io/gorm"
)

type Config struct {
	ID    int `gorm:"primarykey"`
	Class templates.Class
	Type  string
	Value string
}

// TODO remove- migration only
type ConfigDetails struct {
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
	res := make(map[string]any)
	if err := json.Unmarshal([]byte(d.Value), &res); err != nil {
		panic(err)
	}
	return res
}

// detailsFromMap converts map to device details
func detailsFromMap(config map[string]any) (string, error) {
	b, err := json.Marshal(config)
	return string(b), err
}

// Update updates a config's details to the database
func (d *Config) Update(conf map[string]any) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var config Config
		if err := tx.Where(Config{Class: d.Class, ID: d.ID}).First(&config).Error; err != nil {
			return err
		}

		val, err := detailsFromMap(conf)
		if err != nil {
			return err
		}
		d.Value = val

		return tx.Save(&d).Error
	})
}

// PartialUpdate partially updates a config's details to the database
func (d *Config) PartialUpdate(conf map[string]any) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var config Config
		if err := tx.Where(Config{Class: d.Class, ID: d.ID}).First(&config).Error; err != nil {
			return err
		}

		actual := d.detailsAsMap()
		if err := mergo.Merge(&actual, conf, mergo.WithOverride); err != nil {
			return err
		}

		val, err := detailsFromMap(actual)
		if err != nil {
			return err
		}
		d.Value = val

		return tx.Save(&d).Error
	})
}

// Delete deletes a config from the database
func (d *Config) Delete() error {
	return db.Delete(Config{ID: d.ID}).Error
}

var db *gorm.DB

func Init(instance *gorm.DB) error {
	db = instance
	m := db.Migrator()

	for old, new := range map[string]string{
		"devices":        "configs",
		"device_details": "config_details",
	} {
		if m.HasTable(old) {
			if err := m.RenameTable(old, new); err != nil {
				return err
			}
		}
	}

	err := m.AutoMigrate(new(Config))

	if err == nil && m.HasTable("config_details") {
		err = m.AutoMigrate(new(ConfigDetails))

		if err == nil && m.HasConstraint(new(ConfigDetails), "fk_devices_details") {
			err = m.DropConstraint(new(ConfigDetails), "fk_devices_details")
		}
		if err == nil && m.HasColumn(new(ConfigDetails), "device_id") {
			err = m.DropColumn(new(ConfigDetails), "device_id")
		}
	}

	if err == nil && m.HasTable("config_details") {
		var devices []Config
		if err := db.Where(&Config{}).Find(&devices).Error; err != nil {
			return err
		}

		// migrate ConfigDetails into Config.Value
		for _, dev := range devices {
			var details []ConfigDetails
			if err := db.Where(&ConfigDetails{ConfigID: dev.ID}).Find(&details).Error; err != nil {
				return err
			}

			res := make(map[string]any)
			for _, detail := range details {
				res[detail.Key] = detail.Value
			}

			if len(res) > 0 {
				val, err := detailsFromMap(res)
				if err != nil {
					return err
				}
				dev.Value = val

				if err := db.Save(&dev).Error; err != nil {
					return err
				}
			}
		}

		err = m.DropTable("config_details")
	}

	return err
}

// NameForID returns a unique config name for the given id
func NameForID(id int) string {
	return fmt.Sprintf("db:%d", id)
}

// IDForName returns a unique config name for the given id
func IDForName(name string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(name, "db:"))
}

// ConfigurationsByClass returns devices by class from the database
func ConfigurationsByClass(class templates.Class) ([]Config, error) {
	var devices []Config
	tx := db.Where(&Config{Class: class}).Find(&devices)

	// remove devices without details
	res := make([]Config, 0, len(devices))
	for _, dev := range devices {
		if len(dev.Value) > 0 {
			res = append(res, dev)
		}
	}

	return res, tx.Error
}

// ConfigByID returns device by id from the database
func ConfigByID(id int) (Config, error) {
	var config Config
	tx := db.Where(&Config{ID: id}).First(&config)
	return config, tx.Error
}

// AddConfig adds a new config to the database
func AddConfig(class templates.Class, typ string, conf map[string]any) (Config, error) {
	val, err := detailsFromMap(conf)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Class: class,
		Type:  typ,
		Value: val,
	}

	err = db.Create(&config).Error

	return config, err
}
