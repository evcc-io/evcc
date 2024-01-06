module github.com/evcc-io/evcc

go 1.21.1

toolchain go1.21.3

require (
	dario.cat/mergo v1.0.0
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/BurntSushi/toml v1.3.2
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/andig/go-powerwall v0.2.1-0.20230808194509-dd70cdb6e140
	github.com/andig/gosunspec v0.0.0-20231205122018-1daccfa17912
	github.com/andig/mbserver v0.0.0-20230310211055-1d29cbb5820e
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/avast/retry-go/v4 v4.5.1
	github.com/aws/aws-sdk-go v1.49.7
	github.com/basgys/goxml2json v1.1.0
	github.com/basvdlei/gotsmart v0.0.3
	github.com/benbjohnson/clock v1.3.5
	github.com/bogosj/tesla v1.3.1
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/containrrr/shoutrrr v0.8.0
	github.com/coreos/go-oidc/v3 v3.9.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dmarkham/enumer v1.5.9
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/enbility/cemd v0.2.2
	github.com/enbility/eebus-go v0.2.0
	github.com/fatih/structs v1.1.0
	github.com/glebarez/sqlite v1.10.0
	github.com/go-http-utils/etag v0.0.0-20161124023236-513ea8f21eb1
	github.com/go-playground/validator/v10 v10.16.0
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/godbus/dbus/v5 v5.1.1-0.20230522191255-76236955d466
	github.com/gokrazy/updater v0.0.0-20230215172637-813ccc7f21e2
	github.com/golang/mock v1.6.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.5.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gregdel/pushover v1.3.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/grid-x/modbus v0.0.0-20230713135356-d9fefd3ae5a5
	github.com/hashicorp/go-version v1.6.0
	github.com/hasura/go-graphql-client v0.10.1
	github.com/influxdata/influxdb-client-go/v2 v2.13.0
	github.com/insomniacslk/tapo v1.0.0
	github.com/itchyny/gojq v0.12.14
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/copier v0.4.0
	github.com/jinzhu/now v1.1.5
	github.com/joeshaw/carwings v0.0.0-20230425210048-195f7e10e966
	github.com/joho/godotenv v1.5.1
	github.com/jpfielding/go-http-digest v0.0.0-20211006141426-fbc93758452e
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.0.4
	github.com/korylprince/ipnetgen v1.0.1
	github.com/libp2p/zeroconf/v2 v2.2.0
	github.com/lorenzodonini/ocpp-go v0.17.1-0.20231031074144-d214d7c42cbc
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40
	github.com/mabunixda/wattpilot v1.6.2
	github.com/manifoldco/promptui v0.9.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mlnoga/rct v0.1.2-0.20230731074838-03eacb926f99
	github.com/muka/go-bluetooth v0.0.0-20221213043340-85dc80edc4e1
	github.com/mxschmitt/golang-combinations v1.1.0
	github.com/nicksnyder/go-i18n/v2 v2.3.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/philippseith/signalr v0.6.3
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/prometheus-community/pro-bing v0.3.0
	github.com/prometheus/client_golang v1.17.0
	github.com/prometheus/common v0.45.0
	github.com/robertkrimen/otto v0.3.0
	github.com/samber/lo v1.39.0
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.8.4
	github.com/traefik/yaegi v0.15.1
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c
	github.com/volkszaehler/mbmd v0.0.0-20231215091549-af16b1f597b9
	github.com/writeas/go-strip-markdown/v2 v2.1.1
	gitlab.com/bboehmke/sunny v0.16.0
	golang.org/x/crypto/x509roots/fallback v0.0.0-20231218163308-9d2ee975ef9f
	golang.org/x/exp v0.0.0-20231219180239-dc181d75b848
	golang.org/x/net v0.19.0
	golang.org/x/oauth2 v0.15.0
	golang.org/x/sync v0.5.0
	golang.org/x/text v0.14.0
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.5
	nhooyr.io/websocket v1.8.7
	tailscale.com v1.56.1
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/akutz/memconn v0.1.0 // indirect
	github.com/alexbrainman/sspi v0.0.0-20231016080023-1a75b4708caa // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.21.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.42 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.40 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.43 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.38.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.22.0 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/coreos/go-iptables v0.7.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dblohm7/wingoes v0.0.0-20230929194252-e994401fc077 // indirect
	github.com/digitalocean/go-smbios v0.0.0-20180907143718-390a4f403a8e // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fxamacker/cbor/v2 v2.5.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-http-utils/fresh v0.0.0-20161124030543-7231e26a4b27 // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/go-jose/go-jose/v3 v3.0.1 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/nftables v0.1.1-0.20230115205135-9aa6fdf5a28c // indirect
	github.com/gorilla/csrf v1.7.1 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grid-x/serial v0.0.0-20211107191517-583c7356b3aa // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/holoplot/go-avahi v1.0.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/illarion/gonotify v1.0.1 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/insomniacslk/dhcp v0.0.0-20230908212754-65c27093e38a // indirect
	github.com/insomniacslk/xjson v0.0.0-20231023101448-2249e546a131 // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86 // indirect
	github.com/jsimonetti/rtnetlink v1.3.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/kortschak/wol v0.0.0-20200729010619-da482cc4850a // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/sdnotify v1.0.0 // indirect
	github.com/mdlayher/socket v0.5.0 // indirect
	github.com/mergermarket/go-pkcs7 v0.0.0-20170926155232-153b18ea13c9 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.56 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oapi-codegen/runtime v1.0.0 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.4.1-0.20230718164431-9a2bf3000d16 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/relvacode/iso8601 v1.3.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rickb777/date v1.20.1 // indirect
	github.com/rickb777/plural v1.4.1 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/safchain/ethtool v0.3.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tailscale/certstore v0.1.1-0.20231202035212-d3fa0460f47e // indirect
	github.com/tailscale/go-winio v0.0.0-20231025203758-c4f33415bf55 // indirect
	github.com/tailscale/golang-x-crypto v0.0.0-20230713185742-f0b76a10a08e // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/tailscale/web-client-prebuilt v0.0.0-20231213172531-a4fa669015b2 // indirect
	github.com/tailscale/wireguard-go v0.0.0-20231121184858-cc193a0b3272 // indirect
	github.com/tcnksm/go-httpstat v0.2.0 // indirect
	github.com/teivah/onecontext v1.3.0 // indirect
	github.com/u-root/uio v0.0.0-20230305220412-3e8cd9d6bf63 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go4.org/mem v0.0.0-20220726221520-4f986261bf13 // indirect
	go4.org/netipx v0.0.0-20230824141953-6213f710f925 // indirect
	golang.org/x/crypto v0.16.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	gopkg.in/go-playground/validator.v9 v9.30.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gvisor.dev/gvisor v0.0.0-20230928000133-4fe30062272c // indirect
	inet.af/peercred v0.0.0-20210906144145-0893ea02156a // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)

replace github.com/grid-x/modbus => github.com/evcc-io/modbus v0.0.0-20230705160100-23e1df2c01c3

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.16

replace github.com/spf13/viper => github.com/spf13/viper v1.18.1
