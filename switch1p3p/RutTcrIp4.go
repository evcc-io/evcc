package switch1p3p

// LICENSE

// Copyright (c) 2019-2021 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// config elements
type RutTcrIp4 struct {
	conn       UdpReadWrite  // element implementing the UdpReadWrite interface
	udpAddress *net.UDPAddr  // UDP address object
	enabled    bool          // true when enabled, false otherwise
	name       string        // name of the 1p3p switch (used for Hardware-in-the-loop tests)
	phases     int           // current phases setting (1phase or 3 phases)
	cfg        *RutTcrIp4Cfg // config
	log        *util.Logger
}

type UdpReadWrite interface {
	// UdpXchange sends the given command via the given udp connection. Waits max responseTimeout for the response.
	// Returns the response
	UdpXchange(addr *net.UDPAddr, command string, responseTimeoutSec int) (string, error)
}

type RutTcrIp4Cfg struct {
	Url             string // raw url (as string)
	Out1p3p         int    // number of the TcrIp4 output that is used to switch from 1p to 3p and vice versa
	OutReadbackCtrl int    // number of the TcrIp4 output that is used to pulse the readback button. "0" to disable this feature
	OutReadbackSts  int    // number of the TcrIp4 output that reflects the readback status. "0" to disable this feature
	// (toggles with outReadbackCtrl rising edge if we are switched to 3phases, doesn't toggle if 1phase is switched)
	OutWallboxEnable   int // number of the TcrIp4 output that is used to enable/disable the wallbox. "0" to disable this feature
	ResponseTimeoutSec int // UDP response timeout in seconds
	ComCooldownMs      int // Communication cooldown time in ms between two RutTcrIp4 requests (should be at least 100ms)
}

// error types for detailed unit testing
type ErrorType int64

const (
	err_none ErrorType = iota
	err_decode
	err_cfgUrl
	err_udpResolve
	err_udpDialUp
	err_udpSend
	err_udpReceive
	err_setTimeout
	err_cfgOut1p3p
	err_cfgOutReadbackCtrl
	err_cfgOutReadbackSts
	err_cfgOutReadbackCfg
	err_cfgOutWallboxEnable
	err_cfgOutOverlap
	err_cfgRespTimeout
	err_cfgRelaisTimeout
	err_outNumber
	err_responseOut
	err_responseState
	err_targetState
	err_responseLength
	err_readbackState
	err_enableState
)

type Switch1p3pError struct {
	err     error
	errType ErrorType // error type
}

// NewSwitchError creates a switch error instance that holds the error type information
func NewSwitchError(errType ErrorType, err error) *Switch1p3pError {
	switchError := &Switch1p3pError{
		err:     err,
		errType: errType,
	}
	return switchError
}

// Error gives the error stored in the switchError instance
func (e *Switch1p3pError) Error() string {
	return e.err.Error()
}

// ErrorType gives the error type stored in the switchError instance
func (e *Switch1p3pError) ErrorType() ErrorType {
	return e.errType
}

func init() {
	// registry works with lowercase!
	registry.Add("ruttcrip4", NewRutTcrIp4FromConfig)
}

// NewRutTcrIp4FromConfig creates a configurable rutenbeck 1p3p switch
func NewRutTcrIp4FromConfig(other map[string]interface{}) (api.ChargePhases, error) {
	cfg := &RutTcrIp4Cfg{
		ResponseTimeoutSec: 2,   //default timeout for response: 2 seconds
		ComCooldownMs:      100, // default communication cooldown time: 100ms
	}
	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, NewSwitchError(err_decode, err)
	}

	return NewRutTcrIp4(cfg, nil)
}

// NewRutTcrIp4 creates a new configurable rutenbeck 1p3p switch
func NewRutTcrIp4(cfg *RutTcrIp4Cfg, conn UdpReadWrite) (api.ChargePhases, error) {

	trace := util.NewLogger("RutIp4")
	trace.DEBUG.Printf("url: %s, out1p3p: %d, outReadbackCtrl: %d, outReadbackSts: %d, outWallboxEnable: %d, responseTimeoutSec: %d, relaisSwitchTimeMs: %d", cfg.Url, cfg.Out1p3p, cfg.OutReadbackCtrl, cfg.OutReadbackSts, cfg.OutWallboxEnable, cfg.ResponseTimeoutSec, cfg.ComCooldownMs)

	// url must be valid
	parsedUrl, err := url.ParseRequestURI(cfg.Url)
	if err != nil {
		return nil, NewSwitchError(err_cfgUrl, err)
	}

	// output numbers must be within valid range 0 ... disabled, or 1..4 for real outputs
	if cfg.Out1p3p < 1 || cfg.Out1p3p > 4 {
		return nil, NewSwitchError(err_cfgOut1p3p, fmt.Errorf("invalid out1p3p config [%d]. Allowed range: 1..4", cfg.Out1p3p))
	}

	if cfg.OutReadbackCtrl < 0 || cfg.OutReadbackCtrl > 4 {
		return nil, NewSwitchError(err_cfgOutReadbackCtrl, fmt.Errorf("invalid outReadbackCtrl config [%d]. Allowed range: 0..4 (0 disables)", cfg.OutReadbackCtrl))
	}

	if cfg.OutReadbackSts < 0 || cfg.OutReadbackSts > 4 {
		return nil, NewSwitchError(err_cfgOutReadbackSts, fmt.Errorf("invalid outReadbackSts config [%d]. Allowed range: 0..4 (0 disables)", cfg.OutReadbackSts))
	}

	if cfg.OutWallboxEnable < 0 || cfg.OutWallboxEnable > 4 {
		return nil, NewSwitchError(err_cfgOutWallboxEnable, fmt.Errorf("invalid outWallboxEnable config [%d]. Allowed range: 0..4 (0 disables)", cfg.OutWallboxEnable))
	}

	// outReadbackCtrl and Sts must be both enabled or disabled
	if (cfg.OutReadbackCtrl == 0 && cfg.OutReadbackSts != 0) ||
		(cfg.OutReadbackCtrl != 0 && cfg.OutReadbackSts == 0) {
		return nil, NewSwitchError(err_cfgOutReadbackCfg, fmt.Errorf("invalid outReadbackCtrl/Sts config. Both must be disabled or enabled but not mixed disabled/enabled"))
	}

	// outputs numbers may not overlap
	if cfg.Out1p3p == cfg.OutReadbackCtrl || cfg.Out1p3p == cfg.OutReadbackSts || cfg.Out1p3p == cfg.OutWallboxEnable {
		return nil, NewSwitchError(err_cfgOutOverlap, fmt.Errorf("invalid out config. Output numbers may not overlap (except if they are disabled with value 0)"))
	}

	if cfg.OutReadbackCtrl != 0 {
		if cfg.OutReadbackCtrl == cfg.OutReadbackSts || cfg.OutReadbackCtrl == cfg.OutWallboxEnable || cfg.OutReadbackSts == cfg.OutWallboxEnable {
			return nil, NewSwitchError(err_cfgOutOverlap, fmt.Errorf("invalid out config. Output numbers may not overlap (except if they are disabled with value 0)"))
		}
	}

	if cfg.ResponseTimeoutSec <= 0 {
		return nil, NewSwitchError(err_cfgRespTimeout, fmt.Errorf("udpResponseTimeout must be > 0 but configured: %v", cfg.ResponseTimeoutSec))
	}

	if cfg.ComCooldownMs < 10 {
		return nil, NewSwitchError(err_cfgRelaisTimeout, fmt.Errorf("commCooldownMs must be >= 10 but configured: %v", cfg.ComCooldownMs))
	}

	// setup udp address
	udpAddr, err := net.ResolveUDPAddr(parsedUrl.Scheme, parsedUrl.Host)
	if err != nil {
		return nil, NewSwitchError(err_udpResolve, err)
	}

	instance := &RutTcrIp4{
		conn:       conn,
		udpAddress: udpAddr,
		enabled:    false,
		name:       "unnamed",
		phases:     1,
		cfg:        cfg,
		log:        trace,
	}

	if instance.conn == nil {
		instance.conn = instance
	}
	return instance, nil
}

// SimType gives the simulation type of the Switch. "Hil_switch1p3p"
// means that this is a real switch that can be tested as "hardware in the loop"
// component - i.e. a simulator can simulate other components
func (c *RutTcrIp4) SimType() (api.SimType, error) {
	return api.Hil_switch1p3p, nil
}

// SetName sets the name
func (c *RutTcrIp4) SetName(name string) error {
	c.name = name
	return nil
}

// Name gets the name
func (c *RutTcrIp4) Name() (string, error) {
	return c.name, nil
}

// Enable implements the ChargeEnable interface.
// Enables or disables the wallbox. If configured and supported: enables/disables the wallbox
// Is also used for allowing a phases switch (may only be switched when disabled)
func (sw *RutTcrIp4) Enable(enable bool) error {
	sw.log.DEBUG.Printf("RutTcrIp4: Enable:%t", enable)
	sw.enabled = enable
	// if configured: disable wallbox
	if sw.cfg.OutWallboxEnable != 0 {
		var outVal int = 0
		if enable {
			outVal = 1
		}
		if err := sw.SetOutput(sw.cfg.OutWallboxEnable, outVal); err != nil {
			return err
		}
	}

	return nil
}

// Enabled implements the ChargeEnable interface.
// Gives the current enabled status
func (sw *RutTcrIp4) Enabled() (bool, error) {
	return sw.enabled, nil
}

// GetPhases1p3p gives the current phases setting of the switch
func (sw *RutTcrIp4) GetPhases1p3p() (int, error) {
	return sw.phases, nil
}

// Phases1p3p implements the api.ChargePhases interface
func (sw *RutTcrIp4) Phases1p3p(phases int) error {

	sw.log.DEBUG.Printf("RutTcrIp4: SwitchPhases:%d", phases)

	if sw.enabled {
		return NewSwitchError(err_enableState, fmt.Errorf("switch1p3p: phase switching only allowed when disabled, but is enabled"))
	}

	if phases == 3 {
		// switch to 3 phases
		if err := sw.SetOutput(sw.cfg.Out1p3p, 1); err != nil {
			return err
		}
	} else {
		// switch to 1 phase
		if err := sw.SetOutput(sw.cfg.Out1p3p, 0); err != nil {
			return err
		}
	}
	// if configured: check if contactor switched to expected phases
	if sw.cfg.OutReadbackCtrl != 0 {
		if err := sw.CheckReadback(phases); err != nil {
			return err
		}
	}
	sw.phases = phases
	return nil
}

// CheckReadback uses the readback signal and status relais to check if the 1p3p and checks
// if the 1p3p contactor is in the expected state (3 ... 3phases, 1 ... 1 phase)
func (sw *RutTcrIp4) CheckReadback(expectedPhases int) error {

	actPhases := 0
	// don't execute readback if not configured
	if sw.cfg.OutReadbackCtrl == 0 || sw.cfg.OutReadbackSts == 0 {
		return nil
	}

	oldState, err := sw.GetOutputState(sw.cfg.OutReadbackSts)
	if err != nil {
		return err
	}
	// if switched to 3phases the ReadbackSts toggles with every rising
	// edge of the ReadbackCtrl signal
	if err = sw.SetOutput(sw.cfg.OutReadbackCtrl, 0); err != nil {
		return err
	}
	// switching the relais takes some time - we have to wait here until it switched physically
	if err = sw.SetOutput(sw.cfg.OutReadbackCtrl, 1); err != nil {
		return err
	}
	// switching the relais takes some time - we have to wait here until it switched physically
	if err = sw.SetOutput(sw.cfg.OutReadbackCtrl, 0); err != nil {
		return err
	}
	newState, err := sw.GetOutputState(sw.cfg.OutReadbackSts)
	if err != nil {
		return err
	}
	// contactor is switched to 1phase if state didn't toggle
	if newState == oldState {
		actPhases = 1
	} else {
		actPhases = 3
	}

	if expectedPhases != actPhases {
		return NewSwitchError(err_readbackState, fmt.Errorf("1p3p contactor switched to [%dp] expecte [%dp]", actPhases, expectedPhases))
	}

	return nil
}

// SetOutput sets the output to the given targetState
func (sw *RutTcrIp4) SetOutput(outNumber, targetState int) error {

	if outNumber < 1 || outNumber > 4 {
		return NewSwitchError(err_outNumber, fmt.Errorf("SetOutput: Invalid output number [%d] - only 1,2,3,4 allowed", outNumber))
	}

	// to set an output send the UDP command "OUTx 1" - where "x" is the
	// output number, send "OUTx 0" to clear the output
	command := fmt.Sprintf("OUT%d %d", outNumber, targetState)
	// set output
	cmdRes, err := sw.conn.UdpXchange(sw.udpAddress, command, sw.cfg.ResponseTimeoutSec)
	if err != nil {
		return err
	}

	if len(cmdRes) != 7 {
		return NewSwitchError(err_responseLength, fmt.Errorf("SetOutput: responseLength [%d] != expected length 7", len(cmdRes)))
	}

	// extract out number from command result
	// "OUTx =y\r\n"
	respOutNumber, err := strconv.Atoi(cmdRes[3:4])
	if err != nil {
		return NewSwitchError(err_responseOut, err)
	}

	if respOutNumber != outNumber {
		return NewSwitchError(err_responseOut, fmt.Errorf("SetOutput: responseOutNumber [%d] != cmdOUtNumber [%d]", respOutNumber, outNumber))
	}

	// extract out status from command result
	// udp command returns the result status as e.g. "OUTx =y\r\n" where
	// x is the output number (1..4) and y is the output status (0/1)
	outState, err := strconv.Atoi(cmdRes[6:7])
	if err != nil {
		return NewSwitchError(err_responseState, err)
	}
	// check if output is in target state
	if outState != targetState {
		return NewSwitchError(err_targetState, fmt.Errorf("output did not switch to expected state [%d]", targetState))
	}

	// output switched to target state - everything is ok
	return nil
}

// GetOutputState reads the status of one rutenbeck TcrIp4 output
func (sw *RutTcrIp4) GetOutputState(outNumber int) (int, error) {

	if outNumber < 1 || outNumber > 4 {
		return 0, NewSwitchError(err_outNumber, fmt.Errorf("GetOUtputState: Invalid output number [%d] - only 1,2,3,4 allowed", outNumber))
	}
	// command returns e.g. "OUT1 =0" or "OUT1 =1"
	command := fmt.Sprintf("OUT%d ?", outNumber)
	cmdRes, err := sw.conn.UdpXchange(sw.udpAddress, command, sw.cfg.ResponseTimeoutSec)
	if err != nil {
		return -1, err
	}
	// only take the 7th character which reflects the status
	status, err := strconv.Atoi(cmdRes[6:7])
	if err != nil {
		return -1, err
	}

	return status, nil
}

// UdpXchange sends the given string as udp frame and returns the result string received via udp
// the maximum allowed size for the result string is 32 characters.
func (sw *RutTcrIp4) UdpXchange(udpAddress *net.UDPAddr, command string, responseTimeoutSec int) (string, error) {

	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return "", NewSwitchError(err_udpDialUp, err)
	}
	defer conn.Close()

	// send command
	time.Sleep(time.Duration(sw.cfg.ComCooldownMs) * time.Millisecond)
	numBytesWritten, err := conn.Write([]byte(command))
	if err != nil {
		return "", NewSwitchError(err_udpSend, err)
	}
	if numBytesWritten != len(command) {
		return "", NewSwitchError(err_udpSend, fmt.Errorf("failed to send the udp command - sent bytes is too small: %d", numBytesWritten))
	}

	// receive response
	const receiveBufferSize = 32
	receiveBuffer := make([]byte, receiveBufferSize)
	err = conn.SetDeadline(time.Now().Add(time.Duration(responseTimeoutSec) * time.Second))
	if err != nil {
		return "", NewSwitchError(err_setTimeout, err)
	}
	bytesRead, err := conn.Read(receiveBuffer)
	if err != nil {
		return "", NewSwitchError(err_udpReceive, err)
	}
	result := strings.TrimRight(string(receiveBuffer[:bytesRead]), "\r\n")
	return result, nil
}
