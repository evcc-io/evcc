package config

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/server/db"
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
	Type    string
	Title   string `json:"deviceTitle,omitempty" mapstructure:"deviceTitle"`
	Icon    string `json:"deviceIcon,omitempty" mapstructure:"deviceIcon"`
	Product string `json:"deviceProduct,omitempty" mapstructure:"deviceProduct"`
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
	return db.Instance.Transaction(func(tx *gorm.DB) error {
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
	return db.Instance.Delete(Config{ID: d.ID}).Error
}

func Init() error {
	return db.Instance.AutoMigrate(new(Config))
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
	tx := db.Instance.Where(&Config{Class: class}).Find(&devices)

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
	tx := db.Instance.Where(&Config{ID: id}).First(&config)
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

	if err := db.Instance.Create(&config).Error; err != nil {
		return Config{}, err
	}

	return config, nil
}
