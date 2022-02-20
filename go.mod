module github.com/evcc-io/evcc

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/BurntSushi/toml v1.0.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/alvaroloes/enumer v1.1.2
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/avast/retry-go/v3 v3.1.1
	github.com/aws/aws-sdk-go v1.42.35
	github.com/basgys/goxml2json v1.1.0
	github.com/benbjohnson/clock v1.3.0
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bogosj/tesla v1.0.2
	github.com/cjrd/allocate v0.0.0-20191115010018-022b87fe59fc
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/containrrr/shoutrrr v0.5.2
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/deepmap/oapi-codegen v1.9.0 // indirect
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dustin/go-humanize v1.0.0
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/evcc-io/eebus v0.0.0-20211108130022-5536fd4b8fa1
	github.com/fatih/structs v1.1.0
	github.com/foogod/go-powerwall v0.2.0
	github.com/go-ping/ping v0.0.0-20211130115550-779d1e919534
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/godbus/dbus/v5 v5.0.6
	github.com/gokrazy/updater v0.0.0-20211121155532-30ae8cd650ea
	github.com/golang/mock v1.6.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/grandcat/zeroconf v1.0.0
	github.com/gregdel/pushover v1.1.0
	github.com/grid-x/modbus v0.0.0-20220110162222-619e2e635c62
	github.com/hashicorp/go-version v1.4.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12
	github.com/influxdata/influxdb-client-go/v2 v2.6.0
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/itchyny/gojq v0.12.6
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/copier v0.3.4
	github.com/joeshaw/carwings v0.0.0-20210629130626-7ce4ec17db73
	github.com/jpfielding/go-http-digest v0.0.0-20211006141426-fbc93758452e
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.14.1 // indirect
	github.com/koron/go-ssdp v0.0.2
	github.com/korylprince/ipnetgen v1.0.1
	github.com/lorenzodonini/ocpp-go v0.15.0
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.45 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/mlnoga/rct v0.1.2-0.20211011124352-7e995f76b592
	github.com/muka/go-bluetooth v0.0.0-20220219050759-674a63b8741a
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nicksnyder/go-i18n/v2 v2.1.2
	github.com/nirasan/go-oauth-pkce-code-verifier v0.0.0-20170819232839-0fbfe93532da
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.17.0 // indirect
	github.com/philippseith/signalr v0.5.3-0.20211205201131-d57b5a34379a
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/robertkrimen/otto v0.0.0-20211024170158-b87d35c0b86f
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/shurcooL/graphql v0.0.0-20200928012149-18c5c3165e3a
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smartystreets/goconvey v1.7.2 // indirect
	github.com/spf13/afero v1.8.0 // indirect
	github.com/spf13/cobra v1.3.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.9.1
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20220208145932-d2d3cba909f5
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	gitlab.com/bboehmke/sunny v0.15.1-0.20211022160056-2fba1c86ade6
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
	golang.org/x/net v0.0.0-20220114011407-0dd24b26b47d
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7
	golang.org/x/tools v0.1.8 // indirect
	google.golang.org/genproto v0.0.0-20220114231437-d2e6a121cae0 // indirect
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/foogod/go-powerwall => github.com/andig/go-powerwall v0.2.1-0.20220205120646-e5220ad9a9a0
