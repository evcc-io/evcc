package push

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"gorm.io/gorm"
)

// Subscription holds a browser Web Push subscription.
type Subscription struct {
	Endpoint  string `json:"endpoint" gorm:"primarykey"`
	Auth      string `json:"auth"`
	P256dh    string `json:"p256dh"`
	UserAgent string `json:"userAgent,omitempty"`
	CreatedAt time.Time
}

func init() {
	db.Register(func(d *gorm.DB) error {
		return d.AutoMigrate(new(Subscription))
	})
}

// AllSubscriptions returns all stored push subscriptions.
func AllSubscriptions() ([]Subscription, error) {
	var subs []Subscription
	if db.Instance == nil {
		return nil, nil
	}
	return subs, db.Instance.Find(&subs).Error
}

// SubscriptionExists returns true if an endpoint is registered in the DB.
func SubscriptionExists(endpoint string) (bool, error) {
	if db.Instance == nil {
		return false, fmt.Errorf("database not initialized")
	}
	var count int64
	err := db.Instance.Model(&Subscription{}).Where("endpoint = ?", endpoint).Count(&count).Error
	return count > 0, err
}

// SaveSubscription upserts a push subscription.
func SaveSubscription(s Subscription) error {
	if db.Instance == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Instance.Save(&s).Error
}

// DeleteSubscription removes a push subscription by endpoint.
func DeleteSubscription(endpoint string) error {
	if db.Instance == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Instance.Delete(&Subscription{Endpoint: endpoint}).Error
}
