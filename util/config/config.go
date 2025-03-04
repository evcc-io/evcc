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
	return m.AutoMigrate(new(Config))
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
