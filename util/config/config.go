package config

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"gorm.io/gorm"
)

// Config is the database mapping for device configurations
// The device prefix ensures unique namespace
//
// TODO migrate vehicle and loadpoints to this schema
type Config struct {
	ID         int `gorm:"primarykey"`
	Class      templates.Class
	Properties `gorm:"embedded"`
	Data       map[string]any `gorm:"column:value;type:string;serializer:json"`
}

type Properties struct {
	Type        string
	Title       string `json:"deviceTitle,omitempty" mapstructure:"deviceTitle"`
	Icon        string `json:"deviceIcon,omitempty" mapstructure:"deviceIcon"`
	Brand       string `json:"productBrand,omitempty" mapstructure:"productBrand"`
	Description string `json:"productDescription,omitempty" mapstructure:"productDescription"`
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
		Other: maps.Clone(d.Data),
	}
	return res
}

// Typed converts device details to typed config
func (d *Config) Typed() Typed {
	res := Typed{
		Type:  d.Type,
		Other: maps.Clone(d.Data),
	}
	return res
}

func WithProperties(p Properties) func(*Config) {
	return func(d *Config) {
		d.Properties = p
	}
}

// Update updates a config's details to the database
func (d *Config) Update(conf map[string]any, opt ...func(*Config)) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var config Config
		if err := tx.Where(Config{Class: d.Class, ID: d.ID}).First(&config).Error; err != nil {
			return err
		}

		d.Data = conf
		for _, o := range opt {
			o(d)
		}

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
				dev.Data = res

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
		if len(dev.Data) > 0 {
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
func AddConfig(class templates.Class, conf map[string]any, opt ...func(*Config)) (Config, error) {
	config := Config{
		Class: class,
		Data:  conf,
	}

	for _, o := range opt {
		o(&config)
	}

	if err := db.Create(&config).Error; err != nil {
		return Config{}, err
	}

	return config, nil
}
