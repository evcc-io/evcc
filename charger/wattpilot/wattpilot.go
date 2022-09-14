package wattpilot

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/pbkdf2"
)

const (
	MAX_RECONNECT_RETRIES = 5
)

//go:generate go run gen/generate.go

var randomSource = rand.New(rand.NewSource(time.Now().UnixNano()))

type EventFunc func(*websocket.Conn, map[string]interface{})

type Wattpilot struct {
	_currentConnection *websocket.Conn
	_requestId         int
	_name              string
	_hostname          string
	_serial            string
	_version           string
	_manufacturer      string
	_devicetype        string
	_protocol          float64
	_secured           bool
	Reconnect          bool

	_token3           string
	_hashedpassword   string
	_host             string
	_password         string
	_isInitialized    bool
	_status           map[string]interface{}
	_reconnectTimeout int64
	eventHandler      map[string]EventFunc

	connected    chan bool
	initialized  chan bool
	sendResponse chan string
	interrupt    chan os.Signal
	done         chan interface{}
}

func NewWattpilot(host string, password string) *Wattpilot {
	w := &Wattpilot{
		_host:             host,
		_password:         password,
		Reconnect:         true,
		connected:         make(chan bool),
		initialized:       make(chan bool),
		sendResponse:      make(chan string),
		done:              make(chan interface{}),
		interrupt:         make(chan os.Signal),
		_isInitialized:    false,
		_requestId:        1,
		_reconnectTimeout: 2,
	}

	signal.Notify(w.interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	w.eventHandler = map[string]EventFunc{
		"hello":          w.onEventHello,
		"authRequired":   w.onEventAuthRequired,
		"response":       w.onEventResponse,
		"authSuccess":    w.onEventAuthSuccess,
		"authError":      w.onEventAuthError,
		"fullStatus":     w.onEventFullStatus,
		"deltaStatus":    w.onEventDeltaStatus,
		"clearInverters": w.onEventClearInverters,
		"updateInverter": w.onEventUpdateInverter,
	}
	return w

}

func (w *Wattpilot) GetName() string {
	return w._name
}

func (w *Wattpilot) GetSerial() string {
	return w._serial
}

func (w *Wattpilot) GetHost() string {
	return w._host
}

func (w *Wattpilot) IsInitialized() bool {
	return w._isInitialized
}

func hasKey(data map[string]interface{}, key string) bool {
	_, isKnown := data[key]
	return isKnown
}

func merge(ms ...map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func sha256sum(data string) string {
	bs := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", bs)
}

func (w *Wattpilot) getRequestId() int {
	current := w._requestId
	w._requestId += 1
	return current
}

func (w *Wattpilot) onEventHello(connection *websocket.Conn, message map[string]interface{}) {

	if hasKey(message, "hostname") {
		w._hostname = message["hostname"].(string)
	}
	if hasKey(message, "friendly_name") {
		w._name = message["friendly_name"].(string)
	} else {
		w._name = w._hostname
	}
	w._serial = message["serial"].(string)
	if hasKey(message, "version") {
		w._version = message["version"].(string)
	}
	w._manufacturer = message["manufacturer"].(string)
	w._devicetype = message["devicetype"].(string)
	w._protocol = message["protocol"].(float64)
	if hasKey(message, "secured") {
		w._secured = message["secured"].(bool)
	}

	pwd_data := pbkdf2.Key([]byte(w._password), []byte(w._serial), 100000, 256, sha512.New)
	w._hashedpassword = base64.StdEncoding.EncodeToString([]byte(pwd_data))[:32]
}

func randomHexString(n int) string {
	b := make([]byte, (n+2)/2) // can be simplified to n/2 if n is always even

	if _, err := randomSource.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)[1 : n+1]
}

func (w *Wattpilot) onEventAuthRequired(connection *websocket.Conn, message map[string]interface{}) {

	token1 := message["token1"].(string)
	token2 := message["token2"].(string)

	w._token3 = randomHexString(32)
	hash1 := sha256sum(token1 + w._hashedpassword)
	hash := sha256sum(w._token3 + token2 + hash1)
	response := map[string]interface{}{
		"type":   "auth",
		"token3": w._token3,
		"hash":   hash,
	}
	err := w.onSendRepsonse(connection, false, response)
	if err != nil {
		w._isInitialized = false
	}
}

func (w *Wattpilot) onSendRepsonse(connection *websocket.Conn, secured bool, message map[string]interface{}) error {

	if secured {
		msgId := message["requestId"].(int)
		payload, _ := json.Marshal(message)

		mac := hmac.New(sha256.New, []byte(w._hashedpassword))
		mac.Write(payload)
		message = make(map[string]interface{})
		message["type"] = "securedMsg"
		message["data"] = string(payload)
		message["requestId"] = strconv.Itoa(msgId) + "sm"
		message["hmac"] = hex.EncodeToString(mac.Sum(nil))
	}

	data, _ := json.Marshal(message)

	err := connection.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return err
	}
	return nil
}

func (w *Wattpilot) onEventResponse(connection *websocket.Conn, message map[string]interface{}) {
	mType := message["type"].(string)
	success, ok := message["success"]
	if ok && success.(bool) {
		return
	}
	if mType == "response" {
		w.sendResponse <- message["message"].(string)
		return
	}
}

func (w *Wattpilot) onEventAuthSuccess(connection *websocket.Conn, message map[string]interface{}) {
	w.connected <- true
}

func (w *Wattpilot) onEventAuthError(connection *websocket.Conn, message map[string]interface{}) {
	w.connected <- false
}

func (w *Wattpilot) onEventFullStatus(connection *websocket.Conn, message map[string]interface{}) {

	isPartial := message["partial"].(bool)
	status := message["status"].(map[string]interface{})

	w._status = merge(w._status, status)

	if isPartial {
		return
	}
	w.initialized <- true
	w._isInitialized = true
}
func (w *Wattpilot) onEventDeltaStatus(connection *websocket.Conn, message map[string]interface{}) {
	status := message["status"].(map[string]interface{})
	w._status = merge(w._status, status)
}
func (w *Wattpilot) onEventClearInverters(connection *websocket.Conn, message map[string]interface{}) {
	// log.Println(message)
}
func (w *Wattpilot) onEventUpdateInverter(connection *websocket.Conn, message map[string]interface{}) {
	// log.Println(message)
}

func (w *Wattpilot) Connect() (bool, error) {

	socketUrl := "ws://" + w._host + "/ws"

	var err error
	w._currentConnection, _, err = websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		return false, err
	}

	go w.receiveHandler(w._currentConnection)
	go w.loop(w._currentConnection)

	isConnected := <-w.connected
	if !isConnected {
		return false, errors.New("could not connect")
	}

	<-w.initialized

	return true, nil
}

func (w *Wattpilot) loop(conn *websocket.Conn) {

	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 1000):
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, []byte(""))
			if err != nil {
				continue
			}

		case <-w.interrupt:
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			// log.Println("Received SIGINT interrupt signal. Closing all pending connections")
			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				// log.Println("Error during closing websocket:", err)
				return
			}

			select {
			case <-w.done:
				// log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				// log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}

func (w *Wattpilot) receiveHandler(connection *websocket.Conn) {
	defer close(w.done)
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			if w.Reconnect {
				reConnectLoop := 0
				for {
					time.Sleep(time.Second * time.Duration(w._reconnectTimeout))
					isConnected, _ := w.Connect()
					if isConnected {
						break
					}
					reConnectLoop += 1
					if reConnectLoop > MAX_RECONNECT_RETRIES {
						break
					}
				}
			}
			break
		}
		// log.Printf("Received: %s\n", msg)
		data := make(map[string]interface{})
		err = json.Unmarshal(msg, &data)
		if err != nil {
			continue
		}
		msgType, isTypeAvailable := data["type"]
		if !isTypeAvailable {
			continue
		}
		funcCall, isKnown := w.eventHandler[msgType.(string)]
		if !isKnown {
			continue
		}
		// log.Printf("Calling " + msgType.(string))
		funcCall(connection, data)
	}
}

func (w *Wattpilot) GetProperty(name string) (interface{}, error) {
	if !w._isInitialized {
		return nil, errors.New("connection is not valid")
	}
	origName := name
	if v, isKnown := propertyMap[name]; isKnown {
		name = v
	}
	m, post := postProcess[origName]
	if post {
		name = m.key
	}
	if !hasKey(w._status, name) {
		return nil, errors.New("Could not find " + name)
	}
	value := w._status[name]
	if post {
		value, _ = m.f(value)
	}
	return value, nil
}

func (w *Wattpilot) SetProperty(name string, value interface{}) error {
	if !w._isInitialized {
		return errors.New("Connection is not valid")
	}
	if !hasKey(w._status, name) {
		return errors.New("Could not find " + name)
	}

	err := w.sendUpdate(name, value)
	if err != nil {
		return err
	}
	w._status[name] = value
	return nil
}

func (w *Wattpilot) transformValue(value interface{}) interface{} {

	switch value := value.(type) {
	case int:
		return value
	case int64:
		return value
	case float64:
		return value
	}
	in_value := fmt.Sprintf("%v", value)
	if out_value, err := strconv.Atoi(in_value); err == nil {
		return out_value
	}
	if out_value, err := strconv.ParseBool(in_value); err == nil {
		return out_value
	}
	if out_value, err := strconv.ParseFloat(in_value, 64); err == nil {
		return out_value
	}

	return in_value
}

func (w *Wattpilot) sendUpdate(name string, value interface{}) error {

	message := make(map[string]interface{})
	message["type"] = "setValue"
	message["requestId"] = w.getRequestId()
	message["key"] = name
	message["value"] = w.transformValue(value)
	return w.onSendRepsonse(w._currentConnection, w._secured, message)

}

func (w *Wattpilot) Status() (map[string]interface{}, error) {
	if !w._isInitialized {
		return nil, errors.New("connection is not initialzed")
	}

	return w._status, nil
}

func (w *Wattpilot) StatusInfo() {

	fmt.Println("Wattpilot: " + w._name)
	fmt.Println("Serial: " + w._serial)

	fmt.Printf("Car Connected: %v\n", w._status["car"].(float64))
	fmt.Printf("Charge Status %v\n", w._status["alw"].(bool))
	fmt.Printf("Mode: %v\n", w._status["lmo"].(float64))
	fmt.Printf("Power: %v\n\nCharge: ", w._status["amp"].(float64))

	for _, i := range []string{"voltage1", "voltage2", "voltage2"} {
		v, _ := w.GetProperty(i)
		fmt.Printf("%v V, ", v)
	}
	fmt.Printf("\n\t")
	for _, i := range []string{"amps1", "amps2", "amps3"} {
		v, _ := w.GetProperty(i)
		fmt.Printf("%v A, ", v)
	}
	fmt.Printf("\n\t")
	for _, i := range []string{"power1", "power2", "power3"} {
		v, _ := w.GetProperty(i)
		fmt.Printf("%v W, ", v)
	}
	fmt.Println("")
}

func (w *Wattpilot) GetPower() (float64, error) {
	v, err := w.GetProperty("power")
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(v.(string), 64)
}

func (w *Wattpilot) GetCurrents() (float64, float64, float64, error) {
	var currents []float64
	for _, i := range []string{"amps1", "amps2", "amps3"} {
		v, err := w.GetProperty(i)
		if err != nil {
			return -1, -1, -1, err
		}
		fi, err := strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return -1, -1, -1, err
		}

		currents = append(currents, fi)
	}
	return currents[0], currents[1], currents[2], nil
}

func (w *Wattpilot) SetCurrent(current float64) error {
	return w.SetProperty("amp", current)
}

func (w *Wattpilot) GetRFID() (string, error) {
	resp, err := w.GetProperty("cak")
	if err != nil {
		return "", err
	}
	return resp.(string), nil
}
