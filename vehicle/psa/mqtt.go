package psa

import (
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

const (
	MQTT_SERVER      = "ssl://mwa.mpsa.com:8885"
	MQTT_REQ_TOPIC   = "psa/RemoteServices/from/cid/"
	MQTT_RESP_TOPIC  = "psa/RemoteServices/to/cid"
	MQTT_EVENT_TOPIC = "psa/RemoteServices/events/MPHRTServices"
	MQTT_TOKEN_TTL   = 890

	PSA_CORRELATION_DATE_FORMAT = "%Y%m%d%H%M%S%f"
	PSA_DATE_FORMAT             = "%Y-%m-%dT%H:%M:%SZ"
)

// var BRAND = {"com.psa.mym.myopel": {"realm": "clientsB2COpel", "brand_code": "OP", "app_name": "MyOpel"},
//          "com.psa.mym.mypeugeot": {"realm": "clientsB2CPeugeot", "brand_code": "AP", "app_name": "MyPeugeot"},
//          "com.psa.mym.mycitroen": {"realm": "clientsB2CCitroen", "brand_code": "AC", "app_name": "MyCitroen"},
//          "com.psa.mym.myds": {"realm": "clientsB2CDS", "brand_code": "DS", "app_name": "MyDS"},
//          "com.psa.mym.myvauxhall": {"realm": "clientsB2CVauxhall", "brand_code": "VX", "app_name": "MyVauxhall"}
//          }

// MQTT_BRANDCODE = {"AP": "AP",
//                   "AC": "AC",
//                   "DS": "AC",
//                   "VX": "OV",
//                   "OP": "OV"
//                   }

// res2 = requests.post(
// 	f"https://mw-{BRAND[package_name]['brand_code'].lower()}-m2c.mym.awsmpsa.com/api/v1/user",
// 	params={
// 		"culture": apk_parser.culture,
// 		"width": 1080,
// 		"version": APP_VERSION
// 	},
// 	data=json.dumps({"site_code": apk_parser.site_code, "ticket": token}),
// 	headers={
// 		"Connection": "Keep-Alive",
// 		"Content-Type": "application/json;charset=UTF-8",
// 		"Source-Agent": "App-Android",
// 		"Token": token,
// 		"User-Agent": "okhttp/4.8.0",
// 		"Version": APP_VERSION
// 	},
// 	cert=("certs/public.pem", "certs/private.pem"),
// )

// res_dict = res2.json()["success"]
// customer_id = BRAND[package_name]["brand_code"] + "-" + res_dict["id"]

// def get_mqtt_customer_id(self):
// 	brand_code = self.customer_id[:2]
// 	return MQTT_BRANDCODE[brand_code] + self.customer_id[2:]

// def __init__(self, topic, vin, req_parameters, customer_id):
// 	self.customer_id = customer_id
// 	self.topic = MQTT_REQ_TOPIC + self.customer_id + topic
// 	self.vin = vin
// 	self.req_parameters = req_parameters
// 	self.date = datetime.now()
// 	self.data = {}

// def get_message_to_json(self, remote_access_token):
// 	return json.dumps(self.get_message(remote_access_token))

// def get_message(self, remote_access_token):
// 	date = datetime.utcnow()
// 	date_str = date.strftime(PSA_DATE_FORMAT)
// 	self.data = {"access_token": remote_access_token, "customer_id": self.customer_id,
// 			"correlation_id": self.__gen_correlation_id(date), "req_date": date_str, "vin": self.vin,
// 			"req_parameters": self.req_parameters}
// 	return self.data

// def __gen_correlation_id(date):
// 	date_str = date.strftime(PSA_CORRELATION_DATE_FORMAT)[:-3]
// 	uuid_str = str(uuid4()).replace("-", "")
// 	correlation_id = uuid_str + date_str
// 	return correlation_id

type Mqtt struct {
	realm  string
	id     string
	vin    string
	client *mqtt.Client
}

// NewMqtt creates a new vehicle
func NewMqtt(log *util.Logger, identity oauth2.TokenSource, realm, id, vin string) (*Mqtt, error) {
	client, err := mqtt.NewClient(log, "", "", "", "", 1, func(o *paho.ClientOptions) {
		o.AddBroker(MQTT_SERVER)
	})
	if err != nil {
		return nil, err
	}

	v := &Mqtt{
		realm:  realm,
		id:     id,
		vin:    vin,
		client: client,
	}

	v.client.Listen(fmt.Sprintf("%s/%s/#", MQTT_RESP_TOPIC, vin), v.onMessage)
	v.client.Listen(fmt.Sprintf("%s/%s", MQTT_EVENT_TOPIC, vin), v.onMessage)

	return v, nil
}

func (v *Mqtt) onMessage(payload string) {
	fmt.Println("onMessage", string(payload))
}
