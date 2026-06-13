module github.com/evcc-io/evcc

go 1.25.3

require (
	dario.cat/mergo v1.0.2
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/PanterSoft/comlynx-go v0.1.0
	github.com/PuerkitoBio/goquery v1.12.0
	github.com/WulfgarW/sensonet v0.0.7
	github.com/andig/go-powerwall v0.3.0
	github.com/andig/gosunspec v0.0.0-20260523125438-3accc276abc0
	github.com/andig/mbserver v0.0.0-20230310211055-1d29cbb5820e
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/aws/aws-sdk-go-v2 v1.41.11
	github.com/aws/aws-sdk-go-v2/config v1.32.22
	github.com/aws/aws-sdk-go-v2/credentials v1.19.21
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.34.2
	github.com/basgys/goxml2json v1.1.0
	github.com/benbjohnson/clock v1.3.5
	github.com/bogosj/tesla v1.3.2-0.20250818120641-a31b7b6396c9
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cli/browser v1.3.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/coder/websocket v1.8.14
	github.com/coreos/go-oidc/v3 v3.18.0
	github.com/d2r2/go-i2c v0.0.0-20191123181816-73a8a799d6bc
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dylanmei/iso8601 v0.1.0
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/enbility/eebus-go v0.7.0
	github.com/enbility/ship-go v0.6.0
	github.com/enbility/spine-go v0.7.0
	github.com/evcc-io/openapi-mcp v0.6.1-0.20260503092507-6199c7ad3baf
	github.com/evcc-io/optimizer v0.0.0-20260531165648-b5cbfebdaa65
	github.com/evcc-io/rct v0.2.0
	github.com/evcc-io/tesla-proxy-client v0.0.0-20260324063928-151fe10796ae
	github.com/fatih/structs v1.1.0
	github.com/getkin/kin-openapi v0.140.0
	github.com/go-http-utils/etag v0.0.0-20161124023236-513ea8f21eb1
	github.com/go-playground/validator/v10 v10.30.3
	github.com/go-telegram/bot v1.21.0
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/godbus/dbus/v5 v5.2.2
	github.com/gokrazy/updater v0.0.0-20250705135802-db129c40879c
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gosimple/slug v1.15.0
	github.com/gregdel/pushover v1.4.0
	github.com/grid-x/modbus v0.0.0-20260527064858-ef3bed576432
	github.com/hashicorp/go-version v1.9.0
	github.com/hashicorp/yamux v0.1.2
	github.com/hasura/go-graphql-client v0.16.0
	github.com/holoplot/go-evdev v0.0.0-20260504100651-66d1748fe847
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	github.com/insomniacslk/tapo v1.1.0
	github.com/itchyny/gojq v0.12.19
	github.com/jarcoal/httpmock v1.4.1
	github.com/jeremywohl/flatten v1.0.1
	github.com/jinzhu/now v1.1.5
	github.com/joeshaw/carwings v0.0.0-20250704173606-1708e349f36c
	github.com/joho/godotenv v1.5.1
	github.com/jpfielding/go-http-digest v0.0.0-20260421181648-7215c19bbaa3
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/koron/go-ssdp v0.9.0
	github.com/korylprince/ipnetgen v1.0.1
	github.com/libp2p/zeroconf/v2 v2.2.0
	github.com/libtnb/sqlite v1.1.1
	github.com/lorenzodonini/ocpp-go v0.19.0
	github.com/lunixbochs/struc v0.0.0-20241101090106-8d528fa2c543
	github.com/mabunixda/wattpilot v1.8.5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/muka/go-bluetooth v0.0.0-20240701044517-04c4f09c514e
	github.com/nicholas-fedor/shoutrrr v0.16.0
	github.com/nicksnyder/go-i18n/v2 v2.6.1
	github.com/olekukonko/tablewriter v1.1.4
	github.com/philippseith/signalr v0.8.0
	github.com/prometheus-community/pro-bing v0.8.0
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.68.1
	github.com/robertkrimen/otto v0.5.1
	github.com/samber/lo v1.53.0
	github.com/sandrolain/httpcache v1.4.0
	github.com/sethvargo/go-password v0.3.1
	github.com/sirupsen/logrus v1.9.4
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smallnest/chanx v1.2.0
	github.com/spali/go-rscp v0.2.2
	github.com/spf13/cast v1.10.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/teslamotors/vehicle-command v0.4.1
	github.com/tess1o/go-ecoflow v1.1.1-0.20251003083510-2ccc15a17e29
	github.com/traefik/yaegi v0.16.1
	github.com/volkszaehler/mbmd v0.0.0-20260604063429-b7fba192fa3c
	github.com/warthog618/go-gpiocdev v0.9.1
	gitlab.com/bboehmke/sunny v0.17.0
	go.bug.st/serial v1.7.1
	go.uber.org/mock v0.6.0
	go.yaml.in/yaml/v4 v4.0.0-rc.4.0.20260501213337-dee8e44820ca
	golang.org/x/crypto v0.52.0
	golang.org/x/crypto/x509roots/fallback v0.0.0-20260602072539-e2ffffe738fb
	golang.org/x/exp v0.0.0-20260603202125-055de637280b
	golang.org/x/net v0.55.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sync v0.20.0
	golang.org/x/text v0.37.0
	golang.org/x/tools v0.45.0
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gorm.io/gorm v1.31.1
	modernc.org/sqlite v1.51.0
)

require (
	bazil.org/fuse v0.0.0-20180421153158-65cc252bf669 // indirect
	bitbucket.org/creachadair/shell v0.0.6 // indirect
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.121.4 // indirect
	cloud.google.com/go/accessapproval v1.8.7 // indirect
	cloud.google.com/go/accesscontextmanager v1.9.6 // indirect
	cloud.google.com/go/aiplatform v1.93.0 // indirect
	cloud.google.com/go/analytics v0.29.0 // indirect
	cloud.google.com/go/apigateway v1.7.7 // indirect
	cloud.google.com/go/apigeeconnect v1.7.7 // indirect
	cloud.google.com/go/apigeeregistry v0.9.6 // indirect
	cloud.google.com/go/apikeys v0.6.0 // indirect
	cloud.google.com/go/appengine v1.9.7 // indirect
	cloud.google.com/go/area120 v0.9.7 // indirect
	cloud.google.com/go/artifactregistry v1.17.1 // indirect
	cloud.google.com/go/asset v1.21.1 // indirect
	cloud.google.com/go/assuredworkloads v1.12.6 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/automl v1.14.7 // indirect
	cloud.google.com/go/baremetalsolution v1.3.6 // indirect
	cloud.google.com/go/batch v1.12.2 // indirect
	cloud.google.com/go/beyondcorp v1.1.6 // indirect
	cloud.google.com/go/bigquery v1.69.0 // indirect
	cloud.google.com/go/bigtable v1.38.0 // indirect
	cloud.google.com/go/billing v1.20.4 // indirect
	cloud.google.com/go/binaryauthorization v1.9.5 // indirect
	cloud.google.com/go/certificatemanager v1.9.5 // indirect
	cloud.google.com/go/channel v1.20.0 // indirect
	cloud.google.com/go/cloudbuild v1.22.2 // indirect
	cloud.google.com/go/clouddms v1.8.7 // indirect
	cloud.google.com/go/cloudtasks v1.13.6 // indirect
	cloud.google.com/go/compute v1.40.0 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/contactcenterinsights v1.17.3 // indirect
	cloud.google.com/go/container v1.43.0 // indirect
	cloud.google.com/go/containeranalysis v0.14.1 // indirect
	cloud.google.com/go/datacatalog v1.26.0 // indirect
	cloud.google.com/go/dataflow v0.11.0 // indirect
	cloud.google.com/go/dataform v0.12.0 // indirect
	cloud.google.com/go/datafusion v1.8.6 // indirect
	cloud.google.com/go/datalabeling v0.9.6 // indirect
	cloud.google.com/go/dataplex v1.26.0 // indirect
	cloud.google.com/go/dataproc v1.12.0 // indirect
	cloud.google.com/go/dataproc/v2 v2.14.0 // indirect
	cloud.google.com/go/dataqna v0.9.7 // indirect
	cloud.google.com/go/datastore v1.20.0 // indirect
	cloud.google.com/go/datastream v1.14.1 // indirect
	cloud.google.com/go/deploy v1.27.2 // indirect
	cloud.google.com/go/dialogflow v1.69.0 // indirect
	cloud.google.com/go/dlp v1.24.0 // indirect
	cloud.google.com/go/documentai v1.37.0 // indirect
	cloud.google.com/go/domains v0.10.6 // indirect
	cloud.google.com/go/edgecontainer v1.4.3 // indirect
	cloud.google.com/go/errorreporting v0.3.2 // indirect
	cloud.google.com/go/essentialcontacts v1.7.6 // indirect
	cloud.google.com/go/eventarc v1.15.5 // indirect
	cloud.google.com/go/filestore v1.10.2 // indirect
	cloud.google.com/go/firestore v1.18.0 // indirect
	cloud.google.com/go/functions v1.19.6 // indirect
	cloud.google.com/go/gaming v1.10.1 // indirect
	cloud.google.com/go/gkebackup v1.8.0 // indirect
	cloud.google.com/go/gkeconnect v0.12.4 // indirect
	cloud.google.com/go/gkehub v0.15.6 // indirect
	cloud.google.com/go/gkemulticloud v1.5.3 // indirect
	cloud.google.com/go/grafeas v0.3.15 // indirect
	cloud.google.com/go/gsuiteaddons v1.7.7 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/iap v1.11.2 // indirect
	cloud.google.com/go/ids v1.5.6 // indirect
	cloud.google.com/go/iot v1.8.6 // indirect
	cloud.google.com/go/kms v1.22.0 // indirect
	cloud.google.com/go/language v1.14.5 // indirect
	cloud.google.com/go/lifesciences v0.10.6 // indirect
	cloud.google.com/go/logging v1.13.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	cloud.google.com/go/managedidentities v1.7.6 // indirect
	cloud.google.com/go/maps v1.21.1 // indirect
	cloud.google.com/go/mediatranslation v0.9.6 // indirect
	cloud.google.com/go/memcache v1.11.6 // indirect
	cloud.google.com/go/metastore v1.14.7 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	cloud.google.com/go/networkconnectivity v1.17.1 // indirect
	cloud.google.com/go/networkmanagement v1.19.1 // indirect
	cloud.google.com/go/networksecurity v0.10.6 // indirect
	cloud.google.com/go/notebooks v1.12.6 // indirect
	cloud.google.com/go/optimization v1.7.6 // indirect
	cloud.google.com/go/orchestration v1.11.9 // indirect
	cloud.google.com/go/orgpolicy v1.15.0 // indirect
	cloud.google.com/go/osconfig v1.14.6 // indirect
	cloud.google.com/go/oslogin v1.14.6 // indirect
	cloud.google.com/go/phishingprotection v0.9.6 // indirect
	cloud.google.com/go/policytroubleshooter v1.11.6 // indirect
	cloud.google.com/go/privatecatalog v0.10.7 // indirect
	cloud.google.com/go/pubsub v1.49.0 // indirect
	cloud.google.com/go/pubsublite v1.8.2 // indirect
	cloud.google.com/go/recaptchaenterprise v1.3.1 // indirect
	cloud.google.com/go/recaptchaenterprise/v2 v2.20.4 // indirect
	cloud.google.com/go/recommendationengine v0.9.6 // indirect
	cloud.google.com/go/recommender v1.13.5 // indirect
	cloud.google.com/go/redis v1.18.2 // indirect
	cloud.google.com/go/resourcemanager v1.10.6 // indirect
	cloud.google.com/go/resourcesettings v1.8.3 // indirect
	cloud.google.com/go/retail v1.22.0 // indirect
	cloud.google.com/go/run v1.10.1 // indirect
	cloud.google.com/go/scheduler v1.11.7 // indirect
	cloud.google.com/go/secretmanager v1.15.0 // indirect
	cloud.google.com/go/security v1.19.0 // indirect
	cloud.google.com/go/securitycenter v1.37.0 // indirect
	cloud.google.com/go/servicecontrol v1.11.1 // indirect
	cloud.google.com/go/servicedirectory v1.12.6 // indirect
	cloud.google.com/go/servicemanagement v1.8.0 // indirect
	cloud.google.com/go/serviceusage v1.6.0 // indirect
	cloud.google.com/go/shell v1.8.6 // indirect
	cloud.google.com/go/spanner v1.83.0 // indirect
	cloud.google.com/go/speech v1.28.0 // indirect
	cloud.google.com/go/storage v1.55.0 // indirect
	cloud.google.com/go/storagetransfer v1.13.0 // indirect
	cloud.google.com/go/talent v1.8.3 // indirect
	cloud.google.com/go/texttospeech v1.13.0 // indirect
	cloud.google.com/go/tpu v1.8.3 // indirect
	cloud.google.com/go/trace v1.11.6 // indirect
	cloud.google.com/go/translate v1.12.6 // indirect
	cloud.google.com/go/video v1.24.0 // indirect
	cloud.google.com/go/videointelligence v1.12.6 // indirect
	cloud.google.com/go/vision v1.2.0 // indirect
	cloud.google.com/go/vision/v2 v2.9.5 // indirect
	cloud.google.com/go/vmmigration v1.8.6 // indirect
	cloud.google.com/go/vmwareengine v1.3.5 // indirect
	cloud.google.com/go/vpcaccess v1.8.6 // indirect
	cloud.google.com/go/webrisk v1.11.1 // indirect
	cloud.google.com/go/websecurityscanner v1.7.6 // indirect
	cloud.google.com/go/workflows v1.14.2 // indirect
	code.gitea.io/sdk/gitea v0.11.3 // indirect
	codeberg.org/go-fonts/dejavu v0.4.0 // indirect
	codeberg.org/go-fonts/latin-modern v0.4.0 // indirect
	codeberg.org/go-fonts/liberation v0.5.0 // indirect
	codeberg.org/go-fonts/stix v0.3.0 // indirect
	codeberg.org/go-latex/latex v0.1.0 // indirect
	codeberg.org/go-pdf/fpdf v0.10.0 // indirect
	contrib.go.opencensus.io/exporter/aws v0.0.0-20181029163544-2befc13012d0 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.15-0.20230702191903-2de6d2748484 // indirect
	contrib.go.opencensus.io/integrations/ocsql v0.1.4 // indirect
	contrib.go.opencensus.io/resource v0.1.1 // indirect
	dmitri.shuralyov.com/app/changes v0.0.0-20180602232624-0a106ad413e3 // indirect
	dmitri.shuralyov.com/gpu/mtl v0.0.0-20221208032759-85de2813cf6b // indirect
	dmitri.shuralyov.com/html/belt v0.0.0-20180602232347-f7d459c86be0 // indirect
	dmitri.shuralyov.com/service/change v0.0.0-20181023043359-a85b471d5412 // indirect
	dmitri.shuralyov.com/state v0.0.0-20180228185332-28bcc343414c // indirect
	eliasnaur.com/font v0.0.0-20230308162249-dd43949cb42d // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	gioui.org v0.2.0 // indirect
	gioui.org/cpu v0.0.0-20220412190645-f1e9e8c3b1f7 // indirect
	gioui.org/shader v1.0.6 // indirect
	gioui.org/x v0.2.0 // indirect
	git.apache.org/thrift.git v0.0.0-20180902110319-2566ecd5d999 // indirect
	git.sr.ht/~jackmordaunt/go-toast v1.0.0 // indirect
	git.sr.ht/~sbinet/cmpimg v0.1.0 // indirect
	git.sr.ht/~sbinet/gg v0.6.0 // indirect
	git.wow.st/gmp/jni v0.0.0-20210610011705-34026c7e22d0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20240806141605-e8a1dd7889d6 // indirect
	github.com/Azure/azure-amqp-common-go/v2 v2.1.0 // indirect
	github.com/Azure/azure-amqp-common-go/v3 v3.2.3 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/azure-sdk-for-go v30.1.0+incompatible // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity/cache v0.3.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys v0.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus v1.9.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/internal v1.1.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/internal/v2 v2.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/internal/v3 v3.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/managementgroups/armmanagementgroups v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys v1.3.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/internal v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1 // indirect
	github.com/Azure/azure-service-bus-go v0.9.1 // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0 // indirect
	github.com/Azure/go-amqp v1.4.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/mocks v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-extensions-for-go/cache v0.1.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/BurntSushi/xgb v0.0.0-20160522181843-27f122750802 // indirect
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/CloudyKit/jet/v6 v6.2.0 // indirect
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.37.8 // indirect
	github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp v1.5.3 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.29.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/cloudmock v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator v0.53.0 // indirect
	github.com/JohnCGriffin/overflow v0.0.0-20211019200055-46fa312c352c // indirect
	github.com/Joker/hpp v1.0.0 // indirect
	github.com/Joker/jade v1.1.3 // indirect
	github.com/JuulLabs-OSS/cbgo v0.0.1 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Netflix/go-expect v0.0.0-20220104043353-73e0943537d2 // indirect
	github.com/OneOfOne/xxhash v1.2.2 // indirect
	github.com/RaveNoX/go-jsoncommentstrip v1.0.0 // indirect
	github.com/Shopify/goreferrer v0.0.0-20220729165902-8cddb4f5de06 // indirect
	github.com/Shopify/sarama v1.19.0 // indirect
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/XSAM/otelsql v0.39.0 // indirect
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5 // indirect
	github.com/ahmetb/go-linq/v3 v3.2.0 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/ajstarks/deck v0.0.0-20200831202436-30c9fc6549a9 // indirect
	github.com/ajstarks/deck/generate v0.0.0-20210309230005-c3f852c02e19 // indirect
	github.com/ajstarks/svgo v0.0.0-20211024235047-1546f124cd8b // indirect
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/alecthomas/assert/v2 v2.3.0 // indirect
	github.com/alecthomas/kingpin v2.2.6+incompatible // indirect
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.0 // indirect
	github.com/alecthomas/repr v0.2.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/alvaroloes/enumer v1.1.2 // indirect
	github.com/anatol/vmtest v0.0.0-20250627153117-302402d269a6 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/andybalholm/stroke v0.0.0-20221221101821-bd29b49d73f0 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/antihax/optional v1.0.0 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.5.0 // indirect
	github.com/aokoli/goutils v1.0.1 // indirect
	github.com/apache/arrow/go/v10 v10.0.1 // indirect
	github.com/apache/arrow/go/v11 v11.0.0 // indirect
	github.com/apache/arrow/go/v12 v12.0.1 // indirect
	github.com/apache/arrow/go/v14 v14.0.2 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apache/beam v2.32.0+incompatible // indirect
	github.com/apache/thrift v0.17.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/apex/log v1.1.4 // indirect
	github.com/apex/logs v0.0.4 // indirect
	github.com/aphistic/golf v0.0.0-20180712155816-02c07f170c5a // indirect
	github.com/aphistic/sweet v0.2.0 // indirect
	github.com/armon/circbuf v0.0.0-20150827004946-bbbad097214e // indirect
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6 // indirect
	github.com/armon/go-metrics v0.0.0-20180917152333-f0300d1749da // indirect
	github.com/armon/go-radix v0.0.0-20180808171621-7fddfc383310 // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5 // indirect
	github.com/aryann/difflib v0.0.0-20170710044230-e206f873d14a // indirect
	github.com/aws/aws-lambda-go v1.13.3 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.19.5 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.87 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.27 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.20.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.28 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.44.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.26.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.41.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.89.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.1.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.60.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.1 // indirect
	github.com/aws/smithy-go v1.27.0 // indirect
	github.com/aybabtme/rgbterm v0.0.0-20170906152045-cc83f3b3ce59 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymanbagabas/go-udiff v0.2.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/azihsoyn/rijndael256 v0.0.0-20200316065338-d14eefa2b66b // indirect
	github.com/bazelbuild/rules_go v0.49.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/bketelsen/crypt v0.0.4 // indirect
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bmatcuk/doublestar v1.1.1 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/bradfitz/go-smtpd v0.0.0-20170404230938-deb6d6237625 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20250403215159-8d39553ac7cf // indirect
	github.com/bsm/ginkgo/v2 v2.12.0 // indirect
	github.com/bsm/gomega v1.27.10 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23 // indirect
	github.com/bytedance/sonic v1.10.0-rc3 // indirect
	github.com/c-bata/go-prompt v0.2.6 // indirect
	github.com/caarlos0/ctrlc v1.0.0 // indirect
	github.com/caarlos0/env/v11 v11.3.1 // indirect
	github.com/campoy/embedmd v1.0.0 // indirect
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e // indirect
	github.com/casbin/casbin/v2 v2.1.2 // indirect
	github.com/cavaliercoder/go-cpio v0.0.0-20180626203310-925f9528c45e // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/certifi/gocertifi v0.0.0-20200922220541-2c3bb06c6054 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.3.1 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.9.2 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/exp/golden v0.0.0-20240806155701-69247e0abc2a // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.0 // indirect
	github.com/chromedp/cdproto v0.0.0-20230802225258-3cf4e6d46a89 // indirect
	github.com/chromedp/chromedp v0.9.2 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/chzyer/logex v1.2.1 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/chzyer/test v1.0.0 // indirect
	github.com/clbanning/x2j v0.0.0-20191024224557-825249438eec // indirect
	github.com/client9/misspell v0.3.4 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.6.0 // indirect
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/cockroachdb/datadriven v0.0.0-20200714090401-bf6692d28da5 // indirect
	github.com/cockroachdb/errors v1.2.4 // indirect
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/typeurl/v2 v2.2.0 // indirect
	github.com/coocood/freecache v1.2.4 // indirect
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/creack/goselect v0.1.2 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/cronokirby/saferith v0.33.0 // indirect
	github.com/cstockton/go-conv v1.0.0 // indirect
	github.com/d2r2/go-logger v0.0.0-20210606094344-60e9d1233e22 // indirect
	github.com/danieljoos/wincred v1.2.0 // indirect
	github.com/dave/jennifer v1.7.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/devigned/tab v0.1.1 // indirect
	github.com/dgraph-io/badger/v2 v2.2007.4 // indirect
	github.com/dgraph-io/ristretto v0.0.3-0.20200630154024-f66de99634de // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dgryski/go-sip13 v0.0.0-20181026042036-e10d5fee7954 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/djherbis/atime v1.1.0 // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/dmarkham/enumer v1.6.3 // indirect
	github.com/dnaeon/go-vcr v1.2.0 // indirect
	github.com/docker/docker v28.5.1+incompatible // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815 // indirect
	github.com/donovanhide/eventsource v0.0.0-20210830082556-c59027999da0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.7.0 // indirect
	github.com/eapache/channels v1.1.0 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/eclipse/paho.golang v0.23.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/enbility/go-avahi v0.0.0-20240909195612-d5de6b280d7a // indirect
	github.com/enbility/zeroconf/v2 v2.0.0-20240920094356-be1cae74fda6 // indirect
	github.com/envoyproxy/go-control-plane v0.14.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/go-control-plane/ratelimit v0.1.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/ernesto-jimenez/httplogger v0.0.0-20220128121225-117514c3f345 // indirect
	github.com/esiqveland/notify v0.11.0 // indirect
	github.com/etcd-io/gofail v0.0.0-20190801230047-ad7f989257ca // indirect
	github.com/ettle/strcase v0.1.1 // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fatih/gomodifytags v1.17.1-0.20250423142747-f3939df9aa3c // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/flosch/pongo2/v4 v4.0.2 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/franela/goblin v0.0.0-20200105215937-c9ffbefa60db // indirect
	github.com/franela/goreq v0.0.0-20171204163338-bcd34c9993f8 // indirect
	github.com/frankban/quicktest v1.14.6 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fullstorydev/grpcurl v1.8.2 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520 // indirect
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7 // indirect
	github.com/getlantern/golog v0.0.0-20190830074920-4ef2e798c2d7 // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55 // indirect
	github.com/getlantern/ops v0.0.0-20190325191751-d70cb0d6f85f // indirect
	github.com/getlantern/systray v1.2.2 // indirect
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.9.1 // indirect
	github.com/gkampitakis/ciinfo v0.3.2 // indirect
	github.com/gkampitakis/go-diff v1.3.2 // indirect
	github.com/gkampitakis/go-snaps v0.5.15 // indirect
	github.com/gliderlabs/ssh v0.2.2 // indirect
	github.com/go-ble/ble v0.0.0-20240122180141-8c5522f54333 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-fonts/dejavu v0.3.2 // indirect
	github.com/go-fonts/latin-modern v0.3.2 // indirect
	github.com/go-fonts/liberation v0.3.2 // indirect
	github.com/go-fonts/stix v0.2.2 // indirect
	github.com/go-gl/glfw v0.0.0-20190409004039-e6da0acd62b1 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20231223183121-56fa3ac82ce7 // indirect
	github.com/go-http-utils/fresh v0.0.0-20161124030543-7231e26a4b27 // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/go-ini/ini v1.25.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-latex/latex v0.0.0-20231108140139-5c1ce85aa4ea // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-openapi/testify/v2 v2.4.0 // indirect
	github.com/go-pdf/fpdf v0.9.0 // indirect
	github.com/go-playground/assert/v2 v2.2.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-quicktest/qt v1.101.0 // indirect
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-test/deep v1.1.1 // indirect
	github.com/go-text/typesetting v0.0.0-20230803102845-24e03d8b5372 // indirect
	github.com/go-text/typesetting-utils v0.0.0-20230616150549-2a7df14b6a22 // indirect
	github.com/goburrow/serial v0.1.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.3.0 // indirect
	github.com/goccmack/gocc v1.0.2 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gogo/googleapis v1.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gokrazy/gokapi v0.0.0-20251205165548-0927bab199d4 // indirect
	github.com/gokrazy/internal v0.0.0-20251208203110-3c1aa9087c82 // indirect
	github.com/gokrazy/tools v0.0.0-20260109180632-8ed49b4fafc7 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.4.3 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7 // indirect
	github.com/golang/mock v1.7.0-rc.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/golanguzb70/lrucache v1.2.0 // indirect
	github.com/gomarkdown/markdown v0.0.0-20230922112808-5421fefb8386 // indirect
	github.com/gomodule/redigo v1.9.3 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/certificate-transparency-go v1.1.2 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/go-attestation v0.4.4-0.20230613144338-a9b6eb1eb888 // indirect
	github.com/google/go-cmdtest v0.4.1-0.20220921163831-55ab3332a786 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-github/v28 v28.1.1 // indirect
	github.com/google/go-licenses v0.0.0-20210329231322-ce1d9163b77d // indirect
	github.com/google/go-pkcs11 v0.3.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/go-replayers/grpcreplay v1.3.0 // indirect
	github.com/google/go-replayers/httpreplay v1.2.0 // indirect
	github.com/google/go-sev-guest v0.6.1 // indirect
	github.com/google/go-tpm v0.9.6 // indirect
	github.com/google/go-tpm-tools v0.3.13-0.20230620182252-4639ecce2aba // indirect
	github.com/google/go-tspi v0.3.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/licenseclassifier v0.0.0-20210325184830-bb04aff29e72 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible // indirect
	github.com/google/martian/v3 v3.3.3 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/google/renameio v0.1.0 // indirect
	github.com/google/renameio/v2 v2.0.0 // indirect
	github.com/google/rpmpack v0.0.0-20191226140753-aa36bfddb3a0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/safehtml v0.1.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/trillian v1.4.0 // indirect
	github.com/google/wire v0.7.0 // indirect
	github.com/googleapis/cloud-bigtable-clients-test v0.0.3 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/googleapis/go-type-adapters v1.0.0 // indirect
	github.com/googleapis/google-cloud-go-testing v0.0.0-20200911160855-bcd43fbb19e8 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/gordonklaus/ineffassign v0.0.0-20200309095847-7953dde2c7bf // indirect
	github.com/goreleaser/goreleaser v0.134.0 // indirect
	github.com/goreleaser/nfpm v1.2.1 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/gorilla/sessions v1.2.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/grid-x/serial v0.0.0-20211107191517-583c7356b3aa // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/guptarohit/asciigraph v0.7.3 // indirect
	github.com/hamba/avro/v2 v2.17.2 // indirect
	github.com/hanwen/go-fuse/v2 v2.8.0 // indirect
	github.com/hashicorp/consul/api v1.3.0 // indirect
	github.com/hashicorp/consul/sdk v0.3.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.3 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.4 // indirect
	github.com/hashicorp/go-rootcerts v1.0.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.0 // indirect
	github.com/hashicorp/go-syslog v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go.net v0.0.1 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/mdns v1.0.0 // indirect
	github.com/hashicorp/memberlist v0.1.3 // indirect
	github.com/hashicorp/serf v0.8.2 // indirect
	github.com/hazelcast/hazelcast-go-client v1.4.3 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/hinshun/vt10x v0.0.0-20220119200601-820417d04eec // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/hudl/fargo v1.3.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20250417193237-f615e6bd150b // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/insomniacslk/xjson v0.0.0-20231023101448-2249e546a131 // indirect
	github.com/iris-contrib/go.uuid v2.0.0+incompatible // indirect
	github.com/iris-contrib/httpexpect/v2 v2.15.2 // indirect
	github.com/iris-contrib/schema v0.0.6 // indirect
	github.com/itchyny/go-yaml v0.0.0-20251001235044-fca9a0999f15 // indirect
	github.com/itchyny/timefmt-go v0.1.8 // indirect
	github.com/jackc/chunkreader v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgmock v0.0.0-20210724152146-4ad1a8207f65 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3 v1.1.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/pgx/v4 v4.18.3 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jba/templatecheck v0.7.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jellevandenhooff/dkim v0.0.0-20150330215556-f50fe3d243e1 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/jhump/protoreflect v1.9.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmespath/go-jmespath/internal/testify v1.5.1 // indirect
	github.com/jnovack/flag v1.16.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/jordanlewis/gcassert v0.0.0-20250430164644-389ef753e22e // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/joshdk/go-junit v1.0.0 // indirect
	github.com/jpfielding/gowirelog v0.0.0-20200123170752-df8f8dccb721 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/juju/gnuflag v0.0.0-20171113085948-2ce1bb71843d // indirect
	github.com/juju/ratelimit v1.0.1 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/jung-kurt/gofpdf v1.0.3-0.20190309125859-24315acbbda5 // indirect
	github.com/kataras/blocks v0.0.7 // indirect
	github.com/kataras/golog v0.1.9 // indirect
	github.com/kataras/iris/v12 v12.2.6-0.20230908161203-24ba4e8933b9 // indirect
	github.com/kataras/jwt v0.1.10 // indirect
	github.com/kataras/neffos v0.0.22 // indirect
	github.com/kataras/pio v0.0.12 // indirect
	github.com/kataras/sitemap v0.0.6 // indirect
	github.com/kataras/tunnel v0.0.4 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20190725054713-01f96b0aa0cd // indirect
	github.com/keybase/dbus v0.0.0-20220506165403-5aa21ea2c23a // indirect
	github.com/keybase/go-keychain v0.0.1 // indirect
	github.com/kirsle/configdir v0.0.0-20170128060238-e45d2f54772f // indirect
	github.com/kisielk/errcheck v1.5.0 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/knz/go-libedit v1.10.1 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/labstack/echo/v4 v4.11.4 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/ledongthuc/pdf v0.0.0-20220302134840-0c2507a12d80 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/letsencrypt/pkcs11key/v4 v4.0.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lightstep/lightstep-tracer-common/golang/gogo v0.0.0-20190605223551-bc2310a04743 // indirect
	github.com/lightstep/lightstep-tracer-go v0.18.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20251013123823-9fd1530e3ec3 // indirect
	github.com/lunixbochs/vtclean v1.0.0 // indirect
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794 // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/lyft/protoc-gen-star v0.6.1 // indirect
	github.com/lyft/protoc-gen-star/v2 v2.0.4 // indirect
	github.com/lyft/protoc-gen-validate v0.0.13 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mailgun/raymond/v2 v2.0.48 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/maruel/natural v1.1.1 // indirect
	github.com/matryer/try v0.0.0-20161228173917-9ac251b645a2 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mattn/go-shellwords v1.0.10 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/mattn/go-tty v0.0.7 // indirect
	github.com/mattn/go-zglob v0.0.1 // indirect
	github.com/mattn/goveralls v0.0.5 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/maxatome/go-testdeep v1.14.0 // indirect
	github.com/mdelapenya/tlscert v0.2.0 // indirect
	github.com/mediocregopher/radix/v3 v3.8.1 // indirect
	github.com/mergermarket/go-pkcs7 v0.0.0-20170926155232-153b18ea13c9 // indirect
	github.com/mfridman/tparse v0.18.0 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab // indirect
	github.com/microcosm-cc/bluemonday v1.0.25 // indirect
	github.com/microsoft/go-mssqldb v1.9.2 // indirect
	github.com/miekg/dns v1.1.62 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/mitchellh/cli v1.0.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/gox v0.4.0 // indirect
	github.com/mitchellh/iochan v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/sys/mount v0.3.4 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/reexec v0.1.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/modocache/gover v0.0.0-20171022184752-b58185e213c5 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/mwitkow/go-proto-validators v0.2.0 // indirect
	github.com/nats-io/jwt v0.3.2 // indirect
	github.com/nats-io/jwt/v2 v2.8.0 // indirect
	github.com/nats-io/nats-server/v2 v2.12.1 // indirect
	github.com/nats-io/nats.go v1.47.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/neelance/astrewrite v0.0.0-20160511093645-99348263ae86 // indirect
	github.com/neelance/sourcemap v0.0.0-20200213170602-2833bce08e4c // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nishanths/predeclared v0.0.0-20200524104333-86fad755b4d3 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.5.0 // indirect
	github.com/oapi-codegen/runtime v1.1.2 // indirect
	github.com/oasdiff/yaml v0.1.0 // indirect
	github.com/oasdiff/yaml3 v0.0.13 // indirect
	github.com/oklog/oklog v0.3.2 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/olekukonko/ts v0.0.0-20171002115256-78ecb04241c0 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/ginkgo/v2 v2.29.0 // indirect
	github.com/onsi/gomega v1.41.0 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opentracing-contrib/go-observer v0.0.0-20170622124052-a52f23424492 // indirect
	github.com/opentracing/basictracer-go v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.4.5 // indirect
	github.com/openzipkin/zipkin-go v0.2.2 // indirect
	github.com/orisano/pixelmatch v0.0.0-20220722002657-fb0b55479cde // indirect
	github.com/otiai10/copy v1.2.0 // indirect
	github.com/otiai10/curr v1.0.0 // indirect
	github.com/otiai10/mint v1.3.1 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pact-foundation/pact-go v1.0.4 // indirect
	github.com/pascaldekloe/goe v0.0.0-20180627143212-57f6aae5913c // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/paypal/gatt v0.0.0-20151011220935-4ae819d591cf // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/performancecopilot/speed v3.0.0+incompatible // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/peterbourgon/ff v1.2.0 // indirect
	github.com/phpdave11/gofpdf v1.4.2 // indirect
	github.com/phpdave11/gofpdi v1.0.13 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pilebones/go-udev v0.9.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/diff v0.0.0-20210226163009-20ebb0f2a09e // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/profile v1.2.1 // indirect
	github.com/pkg/sftp v1.13.1 // indirect
	github.com/pkg/term v1.2.0-beta.2 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/posener/complete v1.1.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prashantv/gostub v1.1.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/pseudomuto/protoc-gen-doc v1.5.0 // indirect
	github.com/pseudomuto/protokit v0.2.0 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/quic-go/webtransport-go v0.10.0 // indirect
	github.com/raff/goble v0.0.0-20190909174656-72afc67d6a99 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/redis/go-redis/v9 v9.8.0 // indirect
	github.com/relvacode/iso8601 v1.6.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rickb777/date v1.21.1 // indirect
	github.com/rickb777/plural v1.4.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/cors v1.8.0 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/rs/zerolog v1.21.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ruudk/golang-pdf417 v0.0.0-20201230142125-a7e3863a1245 // indirect
	github.com/ryanuber/columnize v0.0.0-20160712163229-9b3edd62028f // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/samuel/go-zookeeper v0.0.0-20190923202752-2cc03de413da // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/sassoftware/go-rpmutils v0.0.0-20190420191620-a8f1baeba37b // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/schollz/closestmatch v2.1.0+incompatible // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shirou/gopsutil/v4 v4.25.10 // indirect
	github.com/shoenig/go-m1cpu v0.1.7 // indirect
	github.com/shoenig/test v1.7.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/shurcooL/component v0.0.0-20170202220835-f88ec8f54cc4 // indirect
	github.com/shurcooL/events v0.0.0-20181021180414-410e4ca65f48 // indirect
	github.com/shurcooL/github_flavored_markdown v0.0.0-20181002035957-2122de532470 // indirect
	github.com/shurcooL/go v0.0.0-20200502201357-93f07166e636 // indirect
	github.com/shurcooL/go-goon v0.0.0-20170922171312-37c2f522c041 // indirect
	github.com/shurcooL/gofontwoff v0.0.0-20180329035133-29b52fc0a18d // indirect
	github.com/shurcooL/gopherjslib v0.0.0-20160914041154-feb6d3990c2c // indirect
	github.com/shurcooL/highlight_diff v0.0.0-20170515013008-09bb4053de1b // indirect
	github.com/shurcooL/highlight_go v0.0.0-20181028180052-98c3abbbae20 // indirect
	github.com/shurcooL/home v0.0.0-20181020052607-80b7ffcb30f9 // indirect
	github.com/shurcooL/htmlg v0.0.0-20170918183704-d01228ac9e50 // indirect
	github.com/shurcooL/httperror v0.0.0-20170206035902-86b7830d14cc // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/httpgzip v0.0.0-20180522190206-b1c53ac65af9 // indirect
	github.com/shurcooL/issues v0.0.0-20181008053335-6292fdc1e191 // indirect
	github.com/shurcooL/issuesapp v0.0.0-20180602232740-048589ce2241 // indirect
	github.com/shurcooL/notifications v0.0.0-20181007000457-627ab5aea122 // indirect
	github.com/shurcooL/octicon v0.0.0-20181028054416-fa4f57f9efb2 // indirect
	github.com/shurcooL/reactions v0.0.0-20181006231557-f2e0b4ca5b82 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/shurcooL/users v0.0.0-20180125191416-49c67e49c537 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/shurcooL/webdavfs v0.0.0-20170829043945-18c3829fa133 // indirect
	github.com/simonvetter/modbus v1.6.0 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/smartystreets/go-aws-auth v0.0.0-20180515143844-0c1422d1fdb9 // indirect
	github.com/smartystreets/goconvey v1.8.1 // indirect
	github.com/smartystreets/gunit v1.0.0 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sony/gobreaker v0.4.1 // indirect
	github.com/sourcegraph/annotate v0.0.0-20160123013949-f4cad6c6324d // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/sourcegraph/syntaxhighlight v0.0.0-20170531221838-bd320f5d308e // indirect
	github.com/spali/go-slicereader v0.0.0-20201122145524-8e262e1a5127 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/speakeasy-api/jsonpath v0.6.0 // indirect
	github.com/speakeasy-api/openapi-overlay v0.10.2 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/spkg/bom v0.0.0-20160624110644-59b7046e48ad // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/streadway/amqp v0.0.0-20190827072141-edfb9018d271 // indirect
	github.com/streadway/handy v0.0.0-20190108123426-d5acb3125c2a // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/suapapa/go_eddystone v1.3.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/substrait-io/substrait-go v0.4.2 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tailscale/depaware v0.0.0-20210622194025-720c4b409502 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07 // indirect
	github.com/tcnksm/go-latest v0.0.0-20170313132115-e3007ae9052e // indirect
	github.com/tdewolff/minify/v2 v2.12.9 // indirect
	github.com/tdewolff/parse/v2 v2.6.8 // indirect
	github.com/tdewolff/test v1.0.9 // indirect
	github.com/teivah/onecontext v1.3.0 // indirect
	github.com/testcontainers/testcontainers-go v0.39.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/memcached v0.39.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/mongodb v0.39.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/nats v0.39.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/redis v0.39.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tj/assert v0.0.0-20171129193455-018094318fb0 // indirect
	github.com/tj/go-elastic v0.0.0-20171221160941-36157cbbebc2 // indirect
	github.com/tj/go-kinesis v0.0.0-20171128231115-08b17f58cb1b // indirect
	github.com/tj/go-spin v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go v1.2.7 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/ulikunitz/xz v0.5.7 // indirect
	github.com/urfave/cli v1.22.4 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/viant/assertly v0.4.8 // indirect
	github.com/viant/toolbox v0.24.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/warthog618/config v0.5.1 // indirect
	github.com/warthog618/go-gpiosim v0.1.1 // indirect
	github.com/warthog618/gpiod v0.8.1 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	github.com/xanzy/go-gitlab v0.31.0 // indirect
	github.com/xanzy/ssh-agent v0.2.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xhit/go-str2duration v1.2.0 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77 // indirect
	github.com/xyproto/randomstring v1.0.5 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yosssi/ace v0.0.5 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/assert v1.3.0 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	github.com/zenazn/goji v0.9.0 // indirect
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a // indirect
	go.einride.tech/aip v0.68.1 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.etcd.io/etcd v0.0.0-20200513171258-e048e166ab9c // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/v2 v2.305.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0 // indirect
	go.etcd.io/etcd/etcdctl/v3 v3.5.0 // indirect
	go.etcd.io/etcd/etcdutl/v3 v3.5.0 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.0 // indirect
	go.etcd.io/etcd/server/v3 v3.5.0 // indirect
	go.etcd.io/etcd/tests/v3 v3.5.0 // indirect
	go.etcd.io/etcd/v3 v3.5.0 // indirect
	go.etcd.io/gofail v0.1.0 // indirect
	go.mongodb.org/mongo-driver v1.17.6 // indirect
	go.mongodb.org/mongo-driver/v2 v2.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/detectors/aws/ec2 v1.37.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.42.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/contrib/propagators/aws v1.37.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.19.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.57.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/oteltest v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/tools v0.0.0-20190618225709-2cfd321de3ee // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go4.org v0.0.0-20180809161055-417644f6feb5 // indirect
	gocloud.dev v0.43.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/build v0.0.0-20190111050920-041ab4dc3f9d // indirect
	golang.org/x/exp/shiny v0.0.0-20241009180824-f66d83c29e7c // indirect
	golang.org/x/exp/typeparams v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/image v0.25.0 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mobile v0.0.0-20231127183840-76ac6878050a // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/perf v0.0.0-20180704124530-6e6d33e29852 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/telemetry v0.0.0-20260508192327-42602be52be6 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools/go/expect v0.1.1-deprecated // indirect
	golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated // indirect
	golang.org/x/tools/gopls v0.21.1 // indirect
	golang.org/x/vuln v1.1.4 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	gonum.org/v1/netlib v0.0.0-20190313105609-8cb42192e0e0 // indirect
	gonum.org/v1/plot v0.15.2 // indirect
	gonum.org/v1/tools v0.0.0-20200318103217-c168b003ce8c // indirect
	google.golang.org/api v0.254.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/genproto/googleapis/bytestream v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.3.0 // indirect
	google.golang.org/grpc/examples v0.0.0-20250407062114-b368379ef8f6 // indirect
	google.golang.org/grpc/gcp/observability v1.0.1 // indirect
	google.golang.org/grpc/security/advancedtls v1.0.0 // indirect
	google.golang.org/grpc/stats/opencensus v1.0.0 // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28 // indirect
	gopkg.in/errgo.v2 v2.1.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.30.0 // indirect
	gopkg.in/inconshreveable/log15.v2 v2.0.0-20180818164646-67afb5ed74ec // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/readline.v1 v1.0.0-20160726135117-62c6fe619375 // indirect
	gopkg.in/resty.v1 v1.12.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.2 // indirect
	gopkg.in/src-d/go-git-fixtures.v3 v3.5.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/sqlite v1.6.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919 // indirect
	honnef.co/go/tools v0.7.0-0.dev.0.20251022135355-8273271481d0 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/cc/v4 v4.28.2 // indirect
	modernc.org/ccgo/v3 v3.17.0 // indirect
	modernc.org/ccgo/v4 v4.34.0 // indirect
	modernc.org/ccorpus v1.11.6 // indirect
	modernc.org/ccorpus2 v1.6.0 // indirect
	modernc.org/ebnf v1.1.0 // indirect
	modernc.org/ebnfutil v1.1.0 // indirect
	modernc.org/fileutil v1.4.0 // indirect
	modernc.org/gc/v2 v2.6.5 // indirect
	modernc.org/gc/v3 v3.1.2 // indirect
	modernc.org/goabi0 v0.2.0 // indirect
	modernc.org/httpfs v1.0.6 // indirect
	modernc.org/lex v1.1.1 // indirect
	modernc.org/lexer v1.0.4 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/opt v0.2.0 // indirect
	modernc.org/scannertest v1.0.2 // indirect
	modernc.org/sortutil v1.2.1 // indirect
	modernc.org/strutil v1.2.1 // indirect
	modernc.org/tcl v1.15.1 // indirect
	modernc.org/token v1.1.0 // indirect
	modernc.org/z v1.7.0 // indirect
	moul.io/http2curl/v2 v2.3.0 // indirect
	mvdan.cc/gofumpt v0.8.0 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
	nullprogram.com/x/optparse v1.0.0 // indirect
	pack.ag/amqp v0.11.2 // indirect
	pgregory.net/rapid v1.1.0 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
	rsc.io/pdf v0.1.1 // indirect
	rsc.io/quote/v3 v3.1.0 // indirect
	rsc.io/sampler v1.3.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
	sourcegraph.com/sourcegraph/appdash v0.0.0-20190731080439-ebfcffb1b5c0 // indirect
	sourcegraph.com/sourcegraph/go-diff v0.5.0 // indirect
	sourcegraph.com/sqs/pbtypes v0.0.0-20180604144634-d3ebe8f20ae4 // indirect
)

tool (
	github.com/dmarkham/enumer
	github.com/evcc-io/evcc/cmd/implement
	github.com/evcc-io/evcc/cmd/openapi
	github.com/evcc-io/openapi-mcp/cmd/openapi-mcp
	github.com/gokrazy/tools/cmd/gok
	go.uber.org/mock/mockgen
	golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize
)

replace github.com/grid-x/modbus => github.com/evcc-io/modbus v0.0.0-20250501165638-8b6f1fbdb7ea

replace github.com/lorenzodonini/ocpp-go => github.com/evcc-io/ocpp-go v0.0.0-20251212212612-b7f92ee0443b
