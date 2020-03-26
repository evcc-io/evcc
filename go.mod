module github.com/andig/evcc

go 1.13

require (
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/benbjohnson/clock v1.0.0
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/golang/mock v1.4.3
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.1
	github.com/gregdel/pushover v0.0.0-20190217183207-15d3fef40636
	github.com/grid-x/modbus v0.0.0-20200108122021-57d05a9f1e1a
	github.com/grid-x/serial v0.0.0-20191104121038-e24bc9bf6f08 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d
	github.com/jsgoecke/tesla v0.0.0-20190206234002-112508e1374e
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mitchellh/mapstructure v1.2.2
	github.com/mjibson/esc v0.2.0
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.6
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.6.2
	github.com/tcnksm/go-latest v0.0.0-20170313132115-e3007ae9052e
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	golang.org/x/sys v0.0.0-20200317113312-5766fd39f98d // indirect
	golang.org/x/tools v0.0.0-20200318150045-ba25ddc85566
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/spf13/viper => github.com/andig/viper v1.6.3-0.20200308172723-deb8393798ec
