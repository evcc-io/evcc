module github.com/evcc-io/evcc

go 1.22.0

require (
	dario.cat/mergo v1.0.0
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/BurntSushi/toml v1.4.0
	github.com/PuerkitoBio/goquery v1.9.2
	github.com/andig/go-powerwall v0.2.1-0.20230808194509-dd70cdb6e140
	github.com/andig/gosunspec v0.0.0-20231205122018-1daccfa17912
	github.com/andig/mbserver v0.0.0-20230310211055-1d29cbb5820e
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/aws/aws-sdk-go v1.54.11
	github.com/basgys/goxml2json v1.1.0
	github.com/basvdlei/gotsmart v0.0.3
	github.com/benbjohnson/clock v1.3.5
	github.com/bogosj/tesla v1.3.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/containrrr/shoutrrr v0.8.0
	github.com/coreos/go-oidc/v3 v3.10.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dmarkham/enumer v1.5.10
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/enbility/eebus-go v0.6.1
	github.com/enbility/ship-go v0.5.2
	github.com/enbility/spine-go v0.6.1
	github.com/evcc-io/tesla-proxy-client v0.0.0-20240221194046-4168b3759701
	github.com/fatih/structs v1.1.0
	github.com/glebarez/sqlite v1.11.0
	github.com/go-http-utils/etag v0.0.0-20161124023236-513ea8f21eb1
	github.com/go-playground/validator/v10 v10.22.0
	github.com/go-sprout/sprout v0.4.1
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/go-viper/mapstructure/v2 v2.0.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gokrazy/updater v0.0.0-20240113102150-4ac511a17e33
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gregdel/pushover v1.3.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/grid-x/modbus v0.0.0-20240503115206-582f2ab60a18
	github.com/hashicorp/go-version v1.7.0
	github.com/hasura/go-graphql-client v0.12.2
	github.com/influxdata/influxdb-client-go/v2 v2.13.0
	github.com/insomniacslk/tapo v1.0.1
	github.com/itchyny/gojq v0.12.16
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/copier v0.4.0
	github.com/jinzhu/now v1.1.5
	github.com/joeshaw/carwings v0.0.0-20240517194654-cf29a185820c
	github.com/joho/godotenv v1.5.1
	github.com/jpfielding/go-http-digest v0.0.0-20240123121450-cffc47d5d6d8
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.4
	github.com/korylprince/ipnetgen v1.0.1
	github.com/libp2p/zeroconf/v2 v2.2.0
	github.com/lorenzodonini/ocpp-go v0.18.0
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/mabunixda/wattpilot v1.8.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mlnoga/rct v0.1.2-0.20240421173556-1c5b75037e2f
	github.com/muka/go-bluetooth v0.0.0-20240115085408-dfdf79b8f61d
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nicksnyder/go-i18n/v2 v2.4.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/philippseith/signalr v0.6.3
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/prometheus-community/pro-bing v0.4.0
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/common v0.55.0
	github.com/robertkrimen/otto v0.4.0
	github.com/samber/lo v1.43.0
	github.com/sirupsen/logrus v1.9.3
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smallnest/chanx v1.2.0
	github.com/spali/go-rscp v0.2.0
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.20.0-alpha.4
	github.com/stretchr/testify v1.9.0
	github.com/teslamotors/vehicle-command v0.0.2
	github.com/traefik/yaegi v0.16.1
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20240611142726-33463eb0324e
	github.com/writeas/go-strip-markdown/v2 v2.1.1
	gitlab.com/bboehmke/sunny v0.16.0
	go.uber.org/mock v0.4.0
	golang.org/x/crypto v0.24.0
	golang.org/x/crypto/x509roots/fallback v0.0.0-20240626151235-a6a393ffd658
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/net v0.26.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sync v0.7.0
	golang.org/x/text v0.16.0
	golang.org/x/tools v0.22.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.10
	nhooyr.io/websocket v1.8.11
)

require (
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/azihsoyn/rijndael256 v0.0.0-20200316065338-d14eefa2b66b // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cstockton/go-conv v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/enbility/zeroconf/v2 v2.0.0-20240210101930-d0004078577b // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-http-utils/fresh v0.0.0-20161124030543-7231e26a4b27 // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/go-jose/go-jose/v4 v4.0.2 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golanguzb70/lrucache v1.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grid-x/serial v0.0.0-20211107191517-583c7356b3aa // indirect
	github.com/holoplot/go-avahi v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/insomniacslk/xjson v0.0.0-20240624131953-2ef5f14e6a74 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mergermarket/go-pkcs7 v0.0.0-20170926155232-153b18ea13c9 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.61 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/relvacode/iso8601 v1.4.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rickb777/date v1.20.6 // indirect
	github.com/rickb777/plural v1.4.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spali/go-slicereader v0.0.0-20201122145524-8e262e1a5127 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/teivah/onecontext v1.3.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240624140628-dc46fd24d27d // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	modernc.org/libc v1.53.4 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.30.1 // indirect
)

replace gopkg.in/yaml.v3 => github.com/andig/yaml v0.0.0-20240531135838-1ff5761ab467

replace github.com/enbility/spine-go => github.com/enbility/spine-go v0.0.0-20240726200332-a983de1e34b8

replace github.com/enbility/ship-go => github.com/enbility/ship-go v0.0.0-20240731093131-37b1302bca66

replace github.com/lorenzodonini/ocpp-go => github.com/evcc-io/ocpp-go v0.0.0-20240730071053-d69e53b0fce9
