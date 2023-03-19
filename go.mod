module github.com/evcc-io/evcc

go 1.20

require (
	github.com/AlecAivazis/survey/v2 v2.3.6
	github.com/BurntSushi/toml v1.2.1
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/andig/gosunspec v0.0.0-20211108155140-af2e73b86e71
	github.com/andig/mbserver v0.0.0-20230310211055-1d29cbb5820e
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/avast/retry-go/v3 v3.1.1
	github.com/aws/aws-sdk-go v1.44.224
	github.com/basgys/goxml2json v1.1.0
	github.com/basvdlei/gotsmart v0.0.3
	github.com/benbjohnson/clock v1.3.0
	github.com/bogosj/tesla v1.1.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cjrd/allocate v0.0.0-20220510215731-986f24f0fb18
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/containrrr/shoutrrr v0.7.1
	github.com/coreos/go-oidc/v3 v3.5.0
	github.com/deepmap/oapi-codegen v1.12.4
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dmarkham/enumer v1.5.8
	github.com/dustin/go-humanize v1.0.1
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/enbility/cemd v0.1.6
	github.com/enbility/eebus-go v0.1.6
	github.com/fatih/structs v1.1.0
	github.com/foogod/go-powerwall v0.2.0
	github.com/glebarez/sqlite v1.7.0
	github.com/go-http-utils/etag v0.0.0-20161124023236-513ea8f21eb1
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gokrazy/updater v0.0.0-20230215172637-813ccc7f21e2
	github.com/golang-module/carbon/v2 v2.2.3
	github.com/golang/mock v1.6.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0
	github.com/gregdel/pushover v1.1.0
	github.com/grid-x/modbus v0.0.0-20230308082654-888f05e1df73
	github.com/hashicorp/go-version v1.6.0
	github.com/hasura/go-graphql-client v0.9.2
	github.com/imdario/mergo v0.3.14
	github.com/influxdata/influxdb-client-go/v2 v2.12.2
	github.com/itchyny/gojq v0.12.12
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/copier v0.3.5
	github.com/joeshaw/carwings v0.0.0-20220711204617-d90801cf2848
	github.com/joho/godotenv v1.5.1
	github.com/jpfielding/go-http-digest v0.0.0-20211006141426-fbc93758452e
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.4
	github.com/korylprince/ipnetgen v1.0.1
	github.com/kr/pretty v0.3.1
	github.com/libp2p/zeroconf/v2 v2.2.0
	github.com/lorenzodonini/ocpp-go v0.16.1-0.20230316233416-0374c843ff52
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/mabunixda/wattpilot v1.2.0
	github.com/manifoldco/promptui v0.9.0
	github.com/mergermarket/go-pkcs7 v0.0.0-20170926155232-153b18ea13c9
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mlnoga/rct v0.1.2-0.20230227143934-71af1fb7dfa1
	github.com/muka/go-bluetooth v0.0.0-20220219050759-674a63b8741a
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nicksnyder/go-i18n/v2 v2.2.1
	github.com/nirasan/go-oauth-pkce-code-verifier v0.0.0-20220510032225-4f9f17eaec4c
	github.com/olekukonko/tablewriter v0.0.5
	github.com/philippseith/signalr v0.6.1-0.20220829124759-bd5ffb679356
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/prometheus-community/pro-bing v0.1.0
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/common v0.42.0
	github.com/robertkrimen/otto v0.2.1
	github.com/samber/lo v1.37.0
	github.com/shurcooL/graphql v0.0.0-20220606043923-3cf50f8a0a29
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cobra v1.6.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.2
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20230312113724-f6764040a78e
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	gitlab.com/bboehmke/sunny v0.15.1-0.20211022160056-2fba1c86ade6
	golang.org/x/exp v0.0.0-20230315142452-642cacee5cc0
	golang.org/x/net v0.8.0
	golang.org/x/oauth2 v0.6.0
	golang.org/x/sync v0.1.0
	golang.org/x/text v0.8.0
	google.golang.org/api v0.114.0
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.24.6
)

require (
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/glebarez/go-sqlite v1.21.0 // indirect
	github.com/go-http-utils/fresh v0.0.0-20161124030543-7231e26a4b27 // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.8.0 // indirect
	github.com/grid-x/serial v0.0.0-20211107191517-583c7356b3aa // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/holoplot/go-avahi v1.0.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.2 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.52 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rickb777/date v1.20.1 // indirect
	github.com/rickb777/plural v1.4.1 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/teivah/onecontext v1.3.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	modernc.org/libc v1.22.3 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.21.0 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace github.com/foogod/go-powerwall => github.com/andig/go-powerwall v0.2.1-0.20230102102528-4d59ac6910c6
