package switch1p3p

import (
	"fmt"
	"net"
	"testing"

	"github.com/evcc-io/evcc/api"
)

type CfgMap map[string]interface{}

type Command struct {
	cmd     string    // expected command
	resp    string    // response to return
	respErr ErrorType // error to return
}

type MockUdpConn struct {
	cmdCnt   int
	commands []Command // commands and responses in the expected order
	//	udpAddrExp *net.UDPAddr // expected udp address
}

func (mockUdp *MockUdpConn) UdpXchange(udpAddress *net.UDPAddr, command string, responseTimeoutSec int) (string, error) {
	if len(mockUdp.commands) == 0 {
		// no commands expected, but called => return error
		return "", NewSwitchError(err_udpSend, fmt.Errorf("Invalid command. Expected: none but was %s", command))
	}
	mockCmd := mockUdp.commands[mockUdp.cmdCnt]
	mockUdp.cmdCnt++

	if mockCmd.cmd != command {
		return "", NewSwitchError(err_udpSend, fmt.Errorf("Invalid command. Expected: %s but was %s", mockCmd.cmd, command))
	}

	if mockCmd.respErr != err_none {
		return mockCmd.resp, NewSwitchError(mockCmd.respErr, fmt.Errorf("UdpXchangeError"))
	} else {
		return mockCmd.resp, nil
	}
}

const (
	cUrlOk = "udp://192.168.1.10:3030"
)

func TestInvalidConfig(t *testing.T) {

	tests := []struct {
		desc       string
		cfgMap     CfgMap
		expErrType ErrorType
	}{
		{
			desc:       "nok: empty url",
			cfgMap:     CfgMap{"url": ""},
			expErrType: err_cfgUrl,
		},
		{
			desc:       "nok: invalid url (udp:// required)",
			cfgMap:     CfgMap{"url": "192.168.1.10:3030"},
			expErrType: err_cfgUrl,
		},
		{
			desc:       "nok: invalid url - wrong: udp//, correct: udp://",
			cfgMap:     CfgMap{"url": "udp//192.168.1.10:3030"},
			expErrType: err_cfgUrl,
		},
		{
			desc:       "nok: out1p3p entry is missing",
			cfgMap:     CfgMap{"url": cUrlOk},
			expErrType: err_cfgOut1p3p,
		},
		{
			desc:       "nok: out1p3p value 0 invalid - must be 1<=value <=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 0},
			expErrType: err_cfgOut1p3p,
		},
		{
			desc:       "nok: out1p3p value -1 invalid - must be 1<=value <=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": -1},
			expErrType: err_cfgOut1p3p,
		},
		{
			desc:       "nok: responseTimeoutSec value 0 invalid - must be >0",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "responseTimeoutSec": 0.0},
			expErrType: err_cfgRespTimeout,
		},
		{
			desc:       "nok: responseTimeoutSec value -0.1 invalid - must be >0",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "responseTimeoutSec": -0.1},
			expErrType: err_cfgRespTimeout,
		},
		{
			desc:       "nok: out2p3p value 5 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 5},
			expErrType: err_cfgOut1p3p,
		},
		{
			desc:       "nok: outReadbackCtrl value -1 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackCtrl": -1},
			expErrType: err_cfgOutReadbackCtrl,
		},
		{
			desc:       "nok: outReadbackCtrl value 5 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackCtrl": 5},
			expErrType: err_cfgOutReadbackCtrl,
		},
		{
			desc:       "nok: outReadbackSts value -1 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": -1},
			expErrType: err_cfgOutReadbackSts,
		},
		{
			desc:       "nok: outReadbackSts value 5 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": 5},
			expErrType: err_cfgOutReadbackSts,
		},
		{
			desc:       "nok: outWallboxEnable value -1 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outWallboxEnable": -1},
			expErrType: err_cfgOutWallboxEnable,
		},
		{
			desc:       "nok: outWallboxEnable value 5 invalid - must be 1<=value<=4",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outWallboxEnable": 5},
			expErrType: err_cfgOutWallboxEnable,
		},
		{
			desc:       "nok: outReadbackSts enabled but outReadbackCtrl not enabled - must be both enabled or both disabled",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": 2},
			expErrType: err_cfgOutReadbackCfg,
		},
		{
			desc:       "nok: outReadbackCtrl enabled but outReadbackSts not enabled - must be both enabled or both disabled",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackCtrl": 2},
			expErrType: err_cfgOutReadbackCfg,
		},
		{
			desc:       "nok: outReadbackCtrl and outReadbackSts have same output number - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": 4, "outReadbackCtrl": 4},
			expErrType: err_cfgOutOverlap,
		},
		{
			desc:       "nok: outReadbackCtrl and out1p3p overlap - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 2, "outReadbackSts": 4, "outReadbackCtrl": 2},
			expErrType: err_cfgOutOverlap,
		},
		{
			desc:       "nok: OutReadbackSts and out1p3p overlap - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 2, "outReadbackSts": 2, "outReadbackCtrl": 3},
			expErrType: err_cfgOutOverlap,
		},
		{
			desc:       "nok: outWallboxEnable and out1p3p overlap - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 2, "outWallboxEnable": 2},
			expErrType: err_cfgOutOverlap,
		},
		{
			desc:       "nok: outWallboxEnable and outReadbackSts overlap - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": 2, "outReadbackCtrl": 3, "outWallboxEnable": 2},
			expErrType: err_cfgOutOverlap,
		},
		{
			desc:       "nok: outWallboxEnable and outReadbackCtrl overlap - must have different output numbers",
			cfgMap:     CfgMap{"url": cUrlOk, "out1p3p": 1, "outReadbackSts": 3, "outReadbackCtrl": 2, "outWallboxEnable": 2},
			expErrType: err_cfgOutOverlap,
		},
	}

	for _, test := range tests {
		t.Logf("%+v", test)
		result, err := NewRutTcrIp4FromConfig(test.cfgMap)
		if result != nil {
			t.Errorf("New with invalid config didn't fail")
		} else if err == nil {
			t.Errorf("New without config didn't return an error")
		} else if _, ok := err.(*Switch1p3pError); !ok {
			t.Errorf("New with invalid config didn't return a Switch1p3pError")
		} else if err.(*Switch1p3pError).ErrorType() != test.expErrType {
			t.Errorf("New with invalid config didn't return the expected error type: was[%d], expected[%d]", err.(*Switch1p3pError).ErrorType(), test.expErrType)
		}
	}
}

func TestPhases1p3p(t *testing.T) {

	// simple switch, no readback, no wallbox enable
	rutCfgNoRbNoWbEn := &RutTcrIp4Cfg{
		Url:                "udp://192.168.0.1:30303",
		Out1p3p:            1,
		ResponseTimeoutSec: 2,
		ComCooldownMs:      100,
	}

	// switch with wallbox enable, no readback
	rutCfgNoRbWithWbEn := &RutTcrIp4Cfg{
		Url:                "udp://192.168.0.1:30303",
		Out1p3p:            1,
		OutWallboxEnable:   4,
		ResponseTimeoutSec: 2,
		ComCooldownMs:      100,
	}

	// switch with readback, no wallbox enable
	rutCfgWithRbNoWbEn := &RutTcrIp4Cfg{
		Url:                "udp://192.168.0.1:30303",
		Out1p3p:            2,
		OutReadbackCtrl:    1,
		OutReadbackSts:     3,
		ResponseTimeoutSec: 2,
		ComCooldownMs:      100,
	}

	// switch with readback and with wallbox enable
	rutCfgWithRbWithWbEn := &RutTcrIp4Cfg{
		Url:                "udp://192.168.0.1:30303",
		Out1p3p:            2,
		OutReadbackCtrl:    1,
		OutReadbackSts:     3,
		OutWallboxEnable:   4,
		ResponseTimeoutSec: 2,
		ComCooldownMs:      100,
	}

	tests := []struct {
		desc       string
		cfg        *RutTcrIp4Cfg
		phases     int //phase to switch to 1 or 3
		udpRw      *MockUdpConn
		expErrType ErrorType // expected response error
	}{
		{
			desc:   "->1p, ok, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK response
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->3p, ok, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK response
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->1p, nok: udpDialUp error, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", respErr: err_udpDialUp}, // NOK error returned
				},
			},
			expErrType: err_udpDialUp,
		},
		{
			desc:   "->1p, nok: udpSend error, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", respErr: err_udpSend}, // NOK error returned
				},
			},
			expErrType: err_udpSend,
		},
		{
			desc:   "->1p, nok: udpReceive error, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", respErr: err_udpReceive}, // NOK error returned
				},
			},
			expErrType: err_udpReceive,
		},
		{
			desc:   "->1p, nok: out not switched, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					// resp: other state than requested
					{cmd: "OUT1 0", resp: "OUT1 =1"}, // NOK response 1 instead of 0
				},
			},
			expErrType: err_targetState,
		},
		{
			desc:   "->3p, nok: out not switched, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					// resp: other state than requested
					{cmd: "OUT1 1", resp: "OUT1 =0"}, // NOK response 0 instead of 1
				},
			},
			expErrType: err_targetState,
		},
		{
			desc:   "->1p, nok: udp response no integer, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					// resp: not an integer
					{cmd: "OUT1 0", resp: "OUT1 =a"}, // NOK response not 0 or 1
				},
			},
			expErrType: err_responseState,
		},
		{
			desc:   "->1p, nok: udp response invalid output number, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					// resp: different output
					{cmd: "OUT1 0", resp: "OUT3 =0"}, // NOK response of out3 instead of out1
				},
			},
			expErrType: err_responseOut,
		},
		{
			desc:   "->1p, nok: udp response empty, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					// resp: empty
					{cmd: "OUT1 0", resp: ""}, // NOK empty response
				},
			},
			expErrType: err_responseLength,
		},
		{
			desc:   "->1p, nok: udp response too short, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", resp: "123456"}, // NOK response too short (must be 7 chars)
				},
			},
			expErrType: err_responseLength,
		},
		{
			desc:   "->1p, nok: udp response too long, no readback, no wallboxEnable",
			cfg:    rutCfgNoRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT1 0", resp: "12345678"}, // NOK response too long (must be 7 chars)
				},
			},
			expErrType: err_responseLength,
		},
		{
			desc:   "->1p, ok, no readback, with wallboxEnable",
			cfg:    rutCfgNoRbWithWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT4 0", resp: "OUT4 =0"}, // OK disable wallbox
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK switch to 1 phase
					{cmd: "OUT4 1", resp: "OUT4 =1"}, // OK enable wallbox
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->1p, nok: wallbox enable not switched off, no readback, with wallboxEnable",
			cfg:    rutCfgNoRbWithWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT4 0", resp: "OUT4 =1"}, // NOK response 1 instead of 0
				},
			},
			expErrType: err_targetState,
		},
		{
			desc:   "->1p, nok: wallbox enable not switched back on, no readback, with wallboxEnable",
			cfg:    rutCfgNoRbWithWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT4 0", resp: "OUT4 =0"}, // OK disable wallbox
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK switch to 1 phase
					{cmd: "OUT4 1", resp: "OUT4 =0"}, // NOK enable wallbox fails (shall be 1 but is 0)
				},
			},
			expErrType: err_targetState,
		},
		{
			desc:   "->1p, ok, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 0", resp: "OUT2 =0"}, // OK switch to 1 phase
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Query readback (current state)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1 (= rising edge)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // SOK et ReadbackCtrl back to 0
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Readback new = old => no toggle => 1p
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->3p, ok, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 1", resp: "OUT2 =1"}, // OK switch to 3 phase
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Query readback (current state)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1 (= rising edge)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl back to 0
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Readback new != old => toggle => 3p
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->3p, nok: no readback toggle, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 1", resp: "OUT2 =1"}, // OK switch to 3 phase
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Query readback (current state)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1 (= rising edge)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl back to 0
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // NOK Readback new == old => no toggle => 1p
				},
			},
			expErrType: err_readbackState,
		},
		{
			desc:   "->1p, nok: readback toggle, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 0", resp: "OUT2 =0"}, // OK switch to 3 phase
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Query readback (current state)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1 (= rising edge)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl back to 0
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // NOK Readback new != old => toggle => 3p
				},
			},
			expErrType: err_readbackState,
		},
		{
			desc:   "->3p, ok readback state 1, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 1", resp: "OUT2 =1"}, // OK switch to 3 phase
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Query readback (current state = 1)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1 (= rising edge)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl back to 0
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Readback new != old => toggle => 3p
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->3p, nok: readback ctrl fails to switch, with readback, no wallboxEnable",
			cfg:    rutCfgWithRbNoWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT2 1", resp: "OUT2 =1"}, // OK switch to 3 phase
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Query readback (current state = 1)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =0"}, // NOK Set ReadbackCtrl to 1 fails
				},
			},
			expErrType: err_targetState,
		},
		{
			desc:   "->1p, ok, with readback, with wallboxEnable",
			cfg:    rutCfgWithRbWithWbEn,
			phases: 1,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT4 0", resp: "OUT4 =0"}, // OK disable wallbox
					{cmd: "OUT2 0", resp: "OUT2 =0"}, // OK switch to 1 phase
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Query readback (current state = 1)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Readback new = old => no toggle => 1p
					{cmd: "OUT4 1", resp: "OUT4 =1"}, // OK enable wallbox
				},
			},
			expErrType: err_none,
		},
		{
			desc:   "->3p, ok, with readback, with wallboxEnable",
			cfg:    rutCfgWithRbWithWbEn,
			phases: 3,
			udpRw: &MockUdpConn{cmdCnt: 0,
				commands: []Command{
					{cmd: "OUT4 0", resp: "OUT4 =0"}, // OK disable wallbox
					{cmd: "OUT2 1", resp: "OUT2 =1"}, // OK switch to 3 phases
					{cmd: "OUT3 ?", resp: "OUT3 =1"}, // OK Query readback (current state = 1)
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT1 1", resp: "OUT1 =1"}, // OK Set ReadbackCtrl to 1
					{cmd: "OUT1 0", resp: "OUT1 =0"}, // OK Set ReadbackCtrl to 0
					{cmd: "OUT3 ?", resp: "OUT3 =0"}, // OK Readback new != old => toggle => 3p
					{cmd: "OUT4 1", resp: "OUT4 =1"}, // OK enable wallbox
				},
			},
			expErrType: err_none,
		},
	}

	for _, test := range tests {
		t.Logf("%+v", test)
		tcrIp4, err := NewRutTcrIp4(test.cfg, test.udpRw)
		// check construction result
		if err != nil {
			t.Errorf("NewRutTcrIp4FromConfig failed unexpected: %s", err.Error())
		} else if tcrIp4 == nil {
			t.Errorf("NewRutTcrIp4FromConfig unexpected return value")
		} else if _, ok := tcrIp4.(*RutTcrIp4); !ok {
			t.Errorf("NewRutTcrIp4FromConfig unexpected return type")
		} else {
			err := tcrIp4.(api.ChargeEnable).Enable(false)
			if err == nil {
				err = tcrIp4.Phases1p3p(test.phases)
				if err == nil {
					err = tcrIp4.(api.ChargeEnable).Enable(true)
				}
			}
			if test.expErrType != err_none {
				// error expected
				if err == nil {
					t.Errorf("Error type should be %d but error was 'nil'", test.expErrType)
				} else if _, ok := err.(*Switch1p3pError); !ok {
					t.Errorf("Phases1p3p didn't return an error of type Switch1p3pError")
				} else if err.(*Switch1p3pError).ErrorType() != test.expErrType {
					t.Errorf("Phases1p3p returned error type: %d but expected: %d. Error: %s", err.(*Switch1p3pError).ErrorType(), test.expErrType, err.Error())
				}
			} else {
				// no error expected
				if err != nil {
					t.Errorf("Phases1p3p returned unexpected error: %s", err.Error())
				}
				// number of commands must be executed
				if len(test.udpRw.commands) != test.udpRw.cmdCnt {
					t.Errorf("Phases1p3p did not execute all expected commands. Expected: [%d] but was: [%d]", len(test.udpRw.commands), test.udpRw.cmdCnt)
				}
			}
		}
	}
}

func TestPhases1p3pEnabled(t *testing.T) {

	// switch with wallbox enable, no readback
	rutCfgNoRbWithWbEn := &RutTcrIp4Cfg{
		Url:                "udp://192.168.0.1:30303",
		Out1p3p:            1,
		OutWallboxEnable:   4,
		ResponseTimeoutSec: 2,
		ComCooldownMs:      100,
	}

	udpRw := &MockUdpConn{cmdCnt: 0,
		commands: []Command{
			{cmd: "OUT4 1", resp: "OUT4 =1"}, // OK enable wallbox
		},
	}
	tcrIp4, err := NewRutTcrIp4(rutCfgNoRbWithWbEn, udpRw)
	// check construction result
	if err != nil {
		t.Errorf("NewRutTcrIp4FromConfig failed unexpected: %s", err.Error())
	} else if tcrIp4 == nil {
		t.Errorf("NewRutTcrIp4FromConfig unexpected return value")
	} else if _, ok := tcrIp4.(*RutTcrIp4); !ok {
		t.Errorf("NewRutTcrIp4FromConfig unexpected return type")
	} else {
		// enable
		err := tcrIp4.(api.ChargeEnable).Enable(true)
		if err == nil {
			err = tcrIp4.Phases1p3p(1)
		} else {
			t.Errorf("failed to enable the switch")
		}
		if err == nil {
			t.Errorf("Error type should be %d but error was 'nil'", err_enableState)
		} else if _, ok := err.(*Switch1p3pError); !ok {
			t.Errorf("Phases1p3p didn't return an error of type Switch1p3pError")
		} else if err.(*Switch1p3pError).ErrorType() != err_enableState {
			t.Errorf("Phases1p3p returned error type: %d but expected: %d. Error: %s", err.(*Switch1p3pError).ErrorType(), err_enableState, err.Error())
		}
	}
}
