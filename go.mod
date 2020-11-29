module github.com/andig/evcc

go 1.13

require (
	github.com/PuerkitoBio/goquery v1.6.0
	github.com/andig/evcc-config v0.0.0-20201116095052-80cf13789787
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/benbjohnson/clock v1.0.3
	github.com/containrrr/shoutrrr v0.0.0-20201117204514-8ab1296a9e1f
	github.com/deepmap/oapi-codegen v1.4.1 // indirect
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/fatih/structs v1.1.0
	github.com/go-ping/ping v0.0.0-20201022122018-3977ed72668a
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/godbus/dbus/v5 v5.0.3
	github.com/golang/mock v1.4.4
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gregdel/pushover v0.0.0-20200416074932-c8ad547caed4
	github.com/grid-x/modbus v0.0.0-20200831145459-cb26bc3b5d3d
	github.com/hashicorp/go-version v1.2.1
	github.com/imdario/mergo v0.3.11
	github.com/influxdata/influxdb-client-go/v2 v2.2.0
	github.com/influxdata/line-protocol v0.0.0-20201012155213-5f565037cbc9 // indirect
	github.com/itchyny/gojq v0.11.2
	github.com/jeremywohl/flatten v1.0.1
	github.com/joeshaw/carwings v0.0.0-20191118152321-61b46581307a
	github.com/jsgoecke/tesla v0.0.0-20200530171421-e02ebd220e5a
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.2
	github.com/korylprince/ipnetgen v1.0.0
	github.com/lorenzodonini/ocpp-go v0.12.1-0.20201122163044-c8e61b6f96d2
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/mitchellh/mapstructure v1.3.3
	github.com/mjibson/esc v0.2.0
	github.com/muka/go-bluetooth v0.0.0-20200928120822-44d49b402aee
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nirasan/go-oauth-pkce-code-verifier v0.0.0-20170819232839-0fbfe93532da
	github.com/olekukonko/tablewriter v0.0.4
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/tcnksm/go-latest v0.0.0-20170313132115-e3007ae9052e
	github.com/thoas/go-funk v0.7.0
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20201115202927-ff826598e117
	golang.org/x/net v0.0.0-20200904194848-62affa334b73
	gopkg.in/ini.v1 v1.62.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

replace github.com/spf13/viper => github.com/andig/viper v1.6.3-0.20201123175942-a5af09afab5b
