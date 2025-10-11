package cmd

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate yaml to database (overwrites db settings)",
	Args:  cobra.ExactArgs(0),
	Run:   runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolP(flagReset, "r", false, flagResetDescription)
}

func runMigrate(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	reset := cmd.Flag(flagReset).Changed

	// TODO remove yaml file
	if reset {
		log.INFO.Println("resetting:")
	} else {
		log.INFO.Println("migrating:")
	}

	log.INFO.Println("- global settings")
	if reset {
		settings.Delete(keys.Interval)
		settings.Delete(keys.SponsorToken)
		settings.Delete(keys.Title)
	} else {
		settings.SetInt(keys.Interval, int64(conf.Interval))
		settings.SetString(keys.SponsorToken, conf.SponsorToken)
		if title, ok := conf.Site["title"].(string); ok {
			settings.SetString(keys.Title, title)
		}
	}

	log.INFO.Println("- network")
	if reset {
		settings.Delete(keys.Network)
	} else {
		_ = settings.SetJson(keys.Network, conf.Network)
	}

	log.INFO.Println("- mqtt")
	if reset {
		settings.Delete(keys.Mqtt)
	} else {
		_ = settings.SetJson(keys.Mqtt, conf.Mqtt)
	}

	log.INFO.Println("- influx")
	if reset {
		settings.Delete(keys.Influx)
	} else {
		_ = settings.SetJson(keys.Influx, conf.Influx)
	}

	log.INFO.Println("- hems")
	if reset {
		settings.Delete(keys.Hems)
	} else if conf.HEMS.Type != "" {
		_ = settings.SetYaml(keys.Hems, conf.HEMS)
	}

	log.INFO.Println("- eebus")
	if reset {
		settings.Delete(keys.EEBus)
	} else if conf.EEBus.Configured() {
		_ = settings.SetYaml(keys.EEBus, conf.EEBus)
	}

	log.INFO.Println("- modbusproxy")
	if reset {
		settings.Delete(keys.ModbusProxy)
	} else if len(conf.ModbusProxy) > 0 {
		_ = settings.SetYaml(keys.ModbusProxy, conf.ModbusProxy)
	}

	log.INFO.Println("- messaging")
	if reset {
		settings.Delete(keys.Messaging)
	} else if len(conf.Messaging.Services) > 0 {
		_ = settings.SetYaml(keys.Messaging, conf.Messaging)
	}

	log.INFO.Println("- tariffs")
	if reset {
		settings.Delete(keys.Tariffs)
	} else if conf.Tariffs.Grid.Type != "" || conf.Tariffs.FeedIn.Type != "" || conf.Tariffs.Co2.Type != "" || conf.Tariffs.Planner.Type != "" {
		_ = settings.SetYaml(keys.Tariffs, conf.Tariffs)
	}

	log.INFO.Println("- device configs")
	meterDbIDs := make(map[string]int) // used to migrate circuits
	if reset {
		// site keys
		settings.Delete(keys.GridMeter)
		settings.Delete(keys.PvMeters)
		settings.Delete(keys.AuxMeters)
		settings.Delete(keys.BatteryMeters)
		settings.Delete(keys.ExtMeters)
		// clear config table
		result := db.Instance.Delete(&config.Config{}, "true")
		log.INFO.Printf("  %d entries deleted", result.RowsAffected)
	} else {
		// migrate meters to devices
		for _, meter := range conf.Meters {
			title, ok := meter.Other["title"].(string)
			if ok {
				delete(meter.Other, "title")
			} else {
				title = meter.Name
			}
			if product, ok := meter.Other["template"].(string); ok {
				properties := config.Properties{Type: meter.Type, Title: title, Product: product}
				cnf, err := config.AddConfig(templates.Meter, meter.Other, config.WithProperties(properties))
				if err != nil {
					log.WARN.Printf("migration of meter failed with error: %s", err)
				} else {
					meterDbIDs[meter.Name] = cnf.ID
					log.INFO.Printf("added meter %s with ID %d to database", meter.Name, cnf.ID)
				}
			}
		}
		// migrate site meter references
		if siteMeters, ok := conf.Site["meters"].(map[string]interface{}); ok {
			//settings.SetString(keys.Title, title)
			log.INFO.Printf("Site meters are: %s", siteMeters)
			meterTypes := []string{"pv", "battery", "ext", "aux"}
			for _, meterType := range meterTypes {
				if meters, ck := siteMeters[meterType]; ck {
					var dbIDs []string
					if meterList, ok := meters.([]interface{}); ok {
						for _, meterName := range meterList {
							if id, found := meterDbIDs[meterName.(string)]; found {
								dbIDs = append(dbIDs, fmt.Sprintf("db:%d", id))
							}
						}
					} else {
						if id, found := meterDbIDs[meters.(string)]; found {
							dbIDs = append(dbIDs, fmt.Sprintf("db:%d", id))
						} else {
							log.WARN.Printf("meter '%s' of type '%s' not found in database", meters.(string), meterType)
						}
					}
					if len(dbIDs) > 0 {
						log.INFO.Printf("Set %sMeters to '%s'", meterType, dbIDs)
						settings.SetString(meterType+"Meters", strings.Join(dbIDs, ","))
					}
				}
			}
			if gridMeter, ok := conf.Site["meters"].(map[string]interface{})["grid"].(string); ok {
				if id, found := meterDbIDs[gridMeter]; found {
					log.INFO.Printf("Set grid meter ('%s') to 'db:%d'", gridMeter, id)
					settings.SetString(keys.GridMeter, fmt.Sprintf("db:%d", id))
				}
			}
		}
	}

	log.INFO.Println("- circuits")
	if reset {
		settings.Delete(keys.Circuits)
	} else if len(conf.Circuits) > 0 {
		// migrate meter names to device references
		for _, circuit := range conf.Circuits {
			if m, ok := circuit.Other["meter"].(string); ok {
				if id, found := meterDbIDs[m]; found {
					circuit.Other["meter"] = fmt.Sprintf("db:%d", id)
					log.INFO.Printf("circuit '%s' meter changed from '%v' to '%v'", circuit.Name, m, circuit.Other["meter"])
				} else {
					log.WARN.Printf("meter '%s' of circuit '%s' not found in database", m, circuit.Name)
				}
			}
		}
		_ = settings.SetYaml(keys.Circuits, conf.Circuits)
	}

	log.INFO.Println("- charger")
	chargerDbIDs := make(map[string]int) // used to migrate loadpoints
	if reset {
		// config table already cleared
	} else if len(conf.Chargers) > 0 {
		for _, charger := range conf.Chargers {
			log.INFO.Printf("migrating charger %s", charger)
			if cnf, err := config.AddConfig(templates.Charger, charger.Other, config.WithProperties(config.Properties{
				Type:    charger.Type,
				Title:   charger.Name,
				Product: charger.Other["template"].(string),
			})); err != nil {
				log.WARN.Printf("migration of charger failed with error: %s", err)
			} else {
				chargerDbIDs[charger.Name] = cnf.ID
				log.INFO.Printf("added charger %s with ID %d to database", charger.Name, cnf.ID)
			}
		}
	}

	log.INFO.Println("- loadpoints")
	if reset {
		// config table already cleared
	} else if len(conf.Loadpoints) > 0 {
		for _, lp := range conf.Loadpoints {
			if c, ok := lp.Other["charger"].(string); ok {
				if id, found := chargerDbIDs[c]; found {
					lp.Other["charger"] = fmt.Sprintf("db:%d", id)
					log.INFO.Printf("loadpoint '%s' charger changed from '%v' to '%v'", lp.Name, c, lp.Other["charger"])
				} else {
					log.WARN.Printf("meter '%s' of circuit '%s' not found in database", c, lp.Name)
				}
			}
			log.INFO.Printf("migrating loadpoint %s", lp)
			title, ok := lp.Other["title"].(string)
			if ok {
				delete(lp.Other, "title")
			} else {
				title = lp.Name
			}
			if _, err := config.AddConfig(templates.Loadpoint, lp.Other, config.WithProperties(config.Properties{
				Type:  lp.Type,
				Title: title,
			})); err != nil {
				log.WARN.Printf("migration of loadpoint failed with error: %s", err)
			}
		}
	}

	log.INFO.Println("migration done")

	// wait for shutdown
	<-shutdownDoneC()
}
