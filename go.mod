module github.com/andig/evcc

go 1.13

require (
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/benbjohnson/clock v1.0.0
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/golang/mock v1.4.3
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/gregdel/pushover v0.0.0-20200330145937-ee607c681498
	github.com/grid-x/modbus v0.0.0-20200108122021-57d05a9f1e1a
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d
	github.com/itchyny/gojq v0.10.1
	github.com/joeshaw/carwings v0.0.0-20191118152321-61b46581307a
	github.com/jsgoecke/tesla v0.0.0-20190206234002-112508e1374e
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mitchellh/mapstructure v1.3.0
	github.com/mjibson/esc v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.6.3
	github.com/tcnksm/go-latest v0.0.0-20170313132115-e3007ae9052e
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/volkszaehler/mbmd v0.0.0-20200429162048-6067eb91c6b3
	golang.org/x/crypto v0.0.0-20200403201458-baeed622b8d8 // indirect
	golang.org/x/net v0.0.0-20200425230154-ff2c4b7c35a0 // indirect
	golang.org/x/sys v0.0.0-20200428200454-593003d681fa // indirect
	golang.org/x/tools v0.0.0-20200428211428-0c9eba77bc32
)

replace github.com/spf13/viper => github.com/andig/viper v1.6.3-0.20200308172723-deb8393798ec
