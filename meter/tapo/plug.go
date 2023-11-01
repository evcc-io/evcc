package tapo

// see https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html
// and
// https://github.com/petretiandrea/plugp100/blob/main/plugp100/protocol/klap_protocol.py

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/google/uuid"
)

func NewPlug(addr netip.Addr, logger *util.Logger) *Plug {
	return &Plug{
		log:          logger,
		Addr:         addr,
		terminalUUID: uuid.New(),
	}
}

func (p *Plug) Handshake(username, password string) error {
	if p.session == nil {
		// try the newer KLAP protocol first
		ks := NewKlapSession(p.log)
		if err := ks.Handshake(p.Addr, username, password); err != nil {
			p.log.TRACE.Printf("KLAP handshake failed, trying passthrough handshake")
			// then try the older passthrough protocol
			ps := NewPassthroughSession(p.log)
			if err := ps.Handshake(p.Addr, username, password); err != nil {
				return fmt.Errorf("passthrough handshake failed: %w", err)
			}
			request, err := NewLoginDeviceRequest(username, password)
			if err != nil {
				return err
			}
			requestBytes, err := json.Marshal(request)
			if err != nil {
				return fmt.Errorf("failed to marshal login_device payload: %w", err)
			}
			p.log.TRACE.Printf("Login request: %s", requestBytes)

			response, err := ps.Request(requestBytes)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			p.log.TRACE.Printf("Login response: %s", response)
			var loginResp LoginDeviceResponse
			if err := json.Unmarshal(response, &loginResp); err != nil {
				return fmt.Errorf("failed to unmarshal JSON response: %w", err)
			}
			if loginResp.ErrorCode != 0 {
				return fmt.Errorf("request failed: %s", loginResp.ErrorCode)
			}
			if loginResp.Result.Token == "" {
				return fmt.Errorf("empty token returned by device")
			}
			ps.token = loginResp.Result.Token
			p.session = ps
		} else {
			p.session = ks
		}
		p.log.TRACE.Printf("Session: %+v", p.session)
	}

	return nil
}

func (p *Plug) GetDeviceInfo() (*DeviceInfo, error) {
	if p.session == nil {
		return nil, fmt.Errorf("not logged in")
	}
	request := NewGetDeviceInfoRequest()
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get_device_info payload: %w", err)
	}
	p.log.TRACE.Printf("GetDeviceInfo request: %s", requestBytes)

	response, err := p.session.Request(requestBytes)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	p.log.TRACE.Printf("GetDeviceInfo response: %s", response)
	var infoResp GetDeviceInfoResponse
	if err := json.Unmarshal(response, &infoResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	if infoResp.ErrorCode != 0 {
		return nil, fmt.Errorf("request failed: %s", infoResp.ErrorCode)
	}
	// decode base64-encoded fields
	decodedSSID, err := base64.StdEncoding.DecodeString(infoResp.Result.SSID)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode SSID: %w", err)
	}
	infoResp.Result.DecodedSSID = string(decodedSSID)

	decodedNickname, err := base64.StdEncoding.DecodeString(infoResp.Result.Nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode Nickname: %w", err)
	}
	infoResp.Result.DecodedNickname = string(decodedNickname)

	return &infoResp.Result, nil
}

func (p *Plug) SetDeviceInfo(deviceOn bool) error {
	if p.session == nil {
		return fmt.Errorf("not logged in")
	}
	request := NewSetDeviceInfoRequest(deviceOn)
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal set_device_info payload: %w", err)
	}
	p.log.TRACE.Printf("SetDeviceInfo request: %s", requestBytes)

	response, err := p.session.Request(requestBytes)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	p.log.TRACE.Printf("SetDeviceInfo response: %s", response)
	var infoResp SetDeviceInfoResponse
	if err := json.Unmarshal(response, &infoResp); err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	if infoResp.ErrorCode != 0 {
		return fmt.Errorf("request failed: %s", infoResp.ErrorCode)
	}
	return nil
}

func (p *Plug) GetDeviceUsage() (*DeviceUsage, error) {
	if p.session == nil {
		return nil, fmt.Errorf("not logged in")
	}
	request := NewGetDeviceUsageRequest()
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get_device_usage payload: %w", err)
	}
	p.log.TRACE.Printf("GetDeviceUsage request: %s", requestBytes)

	response, err := p.session.Request(requestBytes)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	p.log.TRACE.Printf("GetDeviceUsage response: %s", response)
	var usageResp GetDeviceUsageResponse
	if err := json.Unmarshal(response, &usageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	if usageResp.ErrorCode != 0 {
		return nil, fmt.Errorf("request failed: %s", usageResp.ErrorCode)
	}
	return &usageResp.Result, nil
}

func (p *Plug) GetEnergyUsage() (*EnergyUsage, error) {
	if p.session == nil {
		return nil, fmt.Errorf("not logged in")
	}
	request := NewGetEnergyUsageRequest()
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get_energy_usage payload: %w", err)
	}
	p.log.TRACE.Printf("GetEnergyUsage request: %s", requestBytes)

	response, err := p.session.Request(requestBytes)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	p.log.TRACE.Printf("GetEnergyUsage response: %s", response)
	var usageResp GetEnergyUsageResponse
	if err := json.Unmarshal(response, &usageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	if usageResp.ErrorCode != 0 {
		return nil, fmt.Errorf("request failed: %s", usageResp.ErrorCode)
	}
	return &usageResp.Result, nil
}

func (p *Plug) On() error {
	return p.SetDeviceInfo(true)
}

func (p *Plug) Off() error {
	return p.SetDeviceInfo(false)
}

func (p *Plug) IsOn() (bool, error) {
	info, err := p.GetDeviceInfo()
	if err != nil {
		return false, err
	}
	return info.DeviceON, nil
}

func NewLoginDeviceRequest(username, password string) (*LoginDeviceRequest, error) {
	if len(password) > 8 {
		return nil, fmt.Errorf("passwords > 8 characters will not work due to a Tapo firmware bug, see https://github.com/fishbigger/TapoP100/issues/4")
	}
	r := LoginDeviceRequest{
		Method: "login_device",
	}
	tmp := sha1.Sum([]byte(username))
	hexsha := make([]byte, hex.EncodedLen(len(tmp)))
	hex.Encode(hexsha, tmp[:])
	r.Params.Username = base64.StdEncoding.EncodeToString(hexsha)
	r.Params.Password = base64.StdEncoding.EncodeToString([]byte(password))
	r.RequestTimeMils = int(time.Now().UnixMilli())
	return &r, nil
}

func NewGetDeviceInfoRequest() *GetDeviceInfoRequest {
	return &GetDeviceInfoRequest{
		Method:          "get_device_info",
		RequestTimeMils: int(time.Now().UnixMilli()),
	}
}

func NewSetDeviceInfoRequest(deviceOn bool) *SetDeviceInfoRequest {
	r := SetDeviceInfoRequest{
		Method: "set_device_info",
	}
	r.Params.DeviceOn = deviceOn
	return &r
}

func NewGetDeviceUsageRequest() *GetDeviceUsageRequest {
	return &GetDeviceUsageRequest{
		Method:          "get_device_usage",
		RequestTimeMils: int(time.Now().UnixMilli()),
	}
}

func NewGetEnergyUsageRequest() *GetEnergyUsageRequest {
	return &GetEnergyUsageRequest{
		Method:          "get_energy_usage",
		RequestTimeMils: int(time.Now().UnixMilli()),
	}
}
