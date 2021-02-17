module github.com/andig/evcc

go 1.16

require (
	github.com/PuerkitoBio/goquery v1.6.1
	github.com/andig/evcc-config v0.0.0-20210210171605-531c04a6bb59
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/benbjohnson/clock v1.0.3
	github.com/bogosj/tesla v0.0.0-20210211144207-92a50058e036
	github.com/containrrr/shoutrrr v0.4.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/eclipse/paho.mqtt.golang v1.3.1
	github.com/fatih/structs v1.1.0
	github.com/go-ping/ping v0.0.0-20201022122018-3977ed72668a
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/godbus/dbus/v5 v5.0.3
	github.com/gokrazy/updater v0.0.0-20210106211705-4d92b338dd24
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.1.5
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gregdel/pushover v0.0.0-20200416074932-c8ad547caed4
	github.com/grid-x/modbus v0.0.0-20200831145459-cb26bc3b5d3d
	github.com/hashicorp/go-version v1.2.1
	github.com/imdario/mergo v0.3.11
	github.com/influxdata/influxdb-client-go/v2 v2.2.1
	github.com/itchyny/gojq v0.12.1
	github.com/jeremywohl/flatten v1.0.1
	github.com/joeshaw/carwings v0.0.0-20210208214325-dacfdd3d7acc
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.2
	github.com/korylprince/ipnetgen v1.0.0
	github.com/lorenzodonini/ocpp-go v0.12.1-0.20201122163044-c8e61b6f96d2
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/mitchellh/mapstructure v1.4.1
	github.com/muka/go-bluetooth v0.0.0-20201211051136-07f31c601d33
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nirasan/go-oauth-pkce-code-verifier v0.0.0-20170819232839-0fbfe93532da
	github.com/olekukonko/tablewriter v0.0.4
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/thoas/go-funk v0.7.0
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/uhthomas/tesla v0.0.0-20210210215721-a076a03e9349
	github.com/volkszaehler/mbmd v0.0.0-20210117183837-59dcc46d62d4
	golang.org/x/net v0.0.0-20201216054612-986b41b23924
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/spf13/viper => github.com/andig/viper v1.6.3-0.20201123175942-a5af09afab5b
