module github.com/evcc-io/evcc

go 1.24.0

require (
	dario.cat/mergo v1.0.2
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/BurntSushi/toml v1.5.0
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/WulfgarW/sensonet v0.0.4
	github.com/andig/go-powerwall v0.2.1-0.20230808194509-dd70cdb6e140
	github.com/andig/gosunspec v0.0.0-20240918203654-860ce51d602b
	github.com/andig/mbserver v0.0.0-20230310211055-1d29cbb5820e
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/aws/aws-sdk-go v1.55.7
	github.com/basgys/goxml2json v1.1.0
	github.com/basvdlei/gotsmart v0.0.3
	github.com/benbjohnson/clock v1.3.5
	github.com/bogosj/tesla v1.3.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cli/browser v1.3.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/coder/websocket v1.8.13
	github.com/containrrr/shoutrrr v0.8.0
	github.com/coreos/go-oidc/v3 v3.15.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.5.0
	github.com/enbility/eebus-go v0.7.0
	github.com/enbility/ship-go v0.6.0
	github.com/enbility/spine-go v0.7.0
	github.com/evcc-io/openapi-mcp v0.3.1-0.20250702152714-5af774c51f55
	github.com/evcc-io/rct v0.1.2-0.20250315164247-d2f41b161785
	github.com/evcc-io/tesla-proxy-client v0.0.0-20240221194046-4168b3759701
	github.com/fatih/structs v1.1.0
	github.com/getkin/kin-openapi v0.132.0
	github.com/glebarez/sqlite v1.11.0
	github.com/go-http-utils/etag v0.0.0-20161124023236-513ea8f21eb1
	github.com/go-playground/validator/v10 v10.27.0
	github.com/go-telegram/bot v1.16.0
	github.com/go-viper/mapstructure/v2 v2.4.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gokrazy/updater v0.0.0-20250705135802-db129c40879c
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gosimple/slug v1.15.0
	github.com/gregdel/pushover v1.3.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/grid-x/modbus v0.0.0-20250717144429-344b140e33c1
	github.com/hashicorp/go-version v1.7.0
	github.com/hasura/go-graphql-client v0.14.4
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	github.com/insomniacslk/tapo v1.0.1
	github.com/itchyny/gojq v0.12.17
	github.com/jarcoal/httpmock v1.4.0
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/now v1.1.5
	github.com/joeshaw/carwings v0.0.0-20250704173606-1708e349f36c
	github.com/joho/godotenv v1.5.1
	github.com/jpfielding/go-http-digest v0.0.0-20240123121450-cffc47d5d6d8
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.6
	github.com/korylprince/ipnetgen v1.0.1
	github.com/libp2p/zeroconf/v2 v2.2.0
	github.com/lorenzodonini/ocpp-go v0.19.0
	github.com/lunixbochs/struc v0.0.0-20241101090106-8d528fa2c543
	github.com/mabunixda/wattpilot v1.8.5
	github.com/mark3labs/mcp-go v0.36.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muka/go-bluetooth v0.0.0-20240701044517-04c4f09c514e
	github.com/mxschmitt/golang-combinations v1.2.0
	github.com/nicksnyder/go-i18n/v2 v2.6.0
	github.com/olekukonko/tablewriter v1.0.9
	github.com/philippseith/signalr v0.7.0
	github.com/prometheus-community/pro-bing v0.7.0
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.65.0
	github.com/robertkrimen/otto v0.5.1
	github.com/samber/lo v1.51.0
	github.com/sirupsen/logrus v1.9.3
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smallnest/chanx v1.2.0
	github.com/spali/go-rscp v0.2.1
	github.com/spf13/cast v1.9.2
	github.com/spf13/cobra v1.9.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.7
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/teslamotors/vehicle-command v0.3.4
	github.com/traefik/yaegi v0.16.1
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20250608084036-9a13257a4991
	github.com/writeas/go-strip-markdown/v2 v2.1.1
	gitlab.com/bboehmke/sunny v0.16.0
	go.uber.org/mock v0.5.2
	go.yaml.in/yaml/v4 v4.0.0-20250730171608-926dd0b54a90
	golang.org/x/crypto v0.40.0
	golang.org/x/crypto/x509roots/fallback v0.0.0-20250711192710-b903b535d3ef
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792
	golang.org/x/net v0.42.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sync v0.16.0
	golang.org/x/text v0.27.0
	golang.org/x/tools v0.35.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.6
	gorm.io/gorm v1.30.1
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/antihax/optional v1.0.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/azihsoyn/rijndael256 v0.0.0-20200316065338-d14eefa2b66b // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/breml/rootcerts v0.2.21 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/cronokirby/saferith v0.33.0 // indirect
	github.com/cstockton/go-conv v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dmarkham/enumer v1.5.10 // indirect
	github.com/donovanhide/eventsource v0.0.0-20210830082556-c59027999da0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/enbility/go-avahi v0.0.0-20240909195612-d5de6b280d7a // indirect
	github.com/enbility/zeroconf/v2 v2.0.0-20240920094356-be1cae74fda6 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-http-utils/fresh v0.0.0-20161124030543-7231e26a4b27 // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gokrazy/gokapi v0.0.0-20250222080418-e140e9c461d8 // indirect
	github.com/gokrazy/internal v0.0.0-20250526201501-559979153369 // indirect
	github.com/gokrazy/tools v0.0.0-20250601065736-ab76ef531d30 // indirect
	github.com/golanguzb70/lrucache v1.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e // indirect
	github.com/google/renameio/v2 v2.0.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/grid-x/serial v0.0.0-20211107191517-583c7356b3aa // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/insomniacslk/xjson v0.0.0-20240821125711-1236daaf6808 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mergermarket/go-pkcs7 v0.0.0-20170926155232-153b18ea13c9 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.62 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/olekukonko/errors v1.1.0 // indirect
	github.com/olekukonko/ll v0.0.9 // indirect
	github.com/onsi/ginkgo/v2 v2.13.0 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.48.2 // indirect
	github.com/quic-go/webtransport-go v0.8.1-0.20241018022711-4ac2c9250e66 // indirect
	github.com/relvacode/iso8601 v1.6.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rickb777/date v1.21.1 // indirect
	github.com/rickb777/plural v1.4.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spali/go-slicereader v0.0.0-20201122145524-8e262e1a5127 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/teivah/onecontext v1.3.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.66.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.38.2 // indirect
)

tool (
	github.com/dmarkham/enumer
	github.com/evcc-io/evcc/cmd/decorate
	github.com/evcc-io/evcc/cmd/openapi
	github.com/gokrazy/tools/cmd/gok
	go.uber.org/mock/mockgen
)

replace github.com/grid-x/modbus => github.com/evcc-io/modbus v0.0.0-20250501165638-8b6f1fbdb7ea

replace github.com/lorenzodonini/ocpp-go => github.com/evcc-io/ocpp-go v0.0.0-20250322092544-c0c6094051c0
