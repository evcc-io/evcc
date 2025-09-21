package charger

import (
    "fmt"
    "net"
    "strconv"
    "strings"
    "time"

    "github.com/evcc-io/evcc/api"
    "github.com/evcc-io/evcc/util"
)

// AvalonQ charger implementation
type AvalonQ struct {
    host     string
    port     string
    username string
    password string
}

func init() {
    registry.Add("avalonq", NewAvalonQFromConfig)
}

// NewAvalonQFromConfig creates a new AvalonQ charger from generic config
func NewAvalonQFromConfig(other map[string]interface{}) (api.Charger, error) {
    cc := struct {
        Host     string
        Port     string
        Username string
        Password string
    }{
        Port:     "4028",
        Username: "admin",
        Password: "admin",
    }

    if err := util.DecodeOther(other, &cc); err != nil {
        return nil, err
    }

    return NewAvalonQ(cc.Host, cc.Port, cc.Username, cc.Password)
}

// NewAvalonQ creates AvalonQ charger
func NewAvalonQ(host, port, username, password string) (api.Charger, error) {
    if port == "" {
        port = "4028"
    }
    if username == "" {
        username = "admin"
    }
    if password == "" {
        password = "admin"
    }

    c := &AvalonQ{
        host:     host,
        port:     port,
        username: username,
        password: password,
    }

    return c, nil
}

// sendCommand sends a command to the miner via TCP
func (c *AvalonQ) sendCommand(command string) (string, error) {
    address := net.JoinHostPort(c.host, c.port)
    
    conn, err := net.DialTimeout("tcp", address, 10*time.Second)
    if err != nil {
        return "", fmt.Errorf("failed to connect to miner: %v", err)
    }
    defer conn.Close()

    // Send command
    _, err = conn.Write([]byte(command))
    if err != nil {
        return "", fmt.Errorf("failed to send command: %v", err)
    }

    // Read response
    buffer := make([]byte, 8192)
    conn.SetReadDeadline(time.Now().Add(30 * time.Second))
    n, err := conn.Read(buffer)
    if err != nil {
        return "", fmt.Errorf("failed to read response: %v", err)
    }

    return string(buffer[:n]), nil
}

// parseResponse parses the miner response format
func (c *AvalonQ) parseResponse(response string) map[string]string {
    result := make(map[string]string)
    
    // Split by | and ,
    parts := strings.Split(response, "|")
    for _, part := range parts {
        fields := strings.Split(part, ",")
        for _, field := range fields {
            if kv := strings.SplitN(field, "=", 2); len(kv) == 2 {
                result[kv[0]] = kv[1]
            }
        }
    }
    
    return result
}

// Status implements the api.Charger interface
func (c *AvalonQ) Status() (api.ChargeStatus, error) {
    response, err := c.sendCommand("summary")
    if err != nil {
        return api.StatusNone, err
    }

    data := c.parseResponse(response)
    
    // Check if miner is actively mining
    if mhsAv, exists := data["MHS av"]; exists {
        if hashrate, err := strconv.ParseFloat(mhsAv, 64); err == nil && hashrate > 1000 {
            return api.StatusC, nil // Mining actively (equivalent to charging)
        }
    }

    // Check if miner is connected but not mining much
    if elapsed, exists := data["Elapsed"]; exists {
        if elapsedTime, err := strconv.Atoi(elapsed); err == nil && elapsedTime > 0 {
            return api.StatusB, nil // Connected but low/no mining
        }
    }

    return api.StatusA, nil // Not connected or error
}

// Enabled implements the api.Charger interface
func (c *AvalonQ) Enabled() (bool, error) {
    response, err := c.sendCommand("summary")
    if err != nil {
        return false, err
    }

    data := c.parseResponse(response)
    
    // Check if mining is active (hashrate > threshold)
    if mhsAv, exists := data["MHS av"]; exists {
        if hashrate, err := strconv.ParseFloat(mhsAv, 64); err == nil {
            return hashrate > 100, nil // Consider enabled if hashrate > 100 MH/s
        }
    }

    return false, nil
}

// Enable implements the api.Charger interface
func (c *AvalonQ) Enable(enable bool) error {
    if enable {
        // Wake up miner - set softon for immediate activation
        timestamp := time.Now().Unix()
        command := fmt.Sprintf("ascset|0,softon,1:%d", timestamp)
        
        _, err := c.sendCommand(command)
        if err != nil {
            return fmt.Errorf("failed to enable mining: %v", err)
        }
        
        // Optional: Also reboot to ensure clean start
        time.Sleep(2 * time.Second)
        _, err = c.sendCommand("ascset|0,reboot,0")
        return err
        
    } else {
        // Put miner in standby - set softoff for immediate standby
        timestamp := time.Now().Unix()
        command := fmt.Sprintf("ascset|0,softoff,1:%d", timestamp)
        
        _, err := c.sendCommand(command)
        if err != nil {
            return fmt.Errorf("failed to disable mining: %v", err)
        }
        return nil
    }
}

// MaxCurrent implements the api.Charger interface
// Maps current to fan speed (power control proxy)
func (c *AvalonQ) MaxCurrent(current int64) error {
    // Map current (6-32A) to fan speed (15-100%)
    // Higher current = more power = higher fan speed
    var fanSpeed int64
    
    if current <= 0 {
        fanSpeed = -1 // Auto fan control
    } else if current <= 6 {
        fanSpeed = 15 // Minimum fan speed
    } else if current >= 32 {
        fanSpeed = 100 // Maximum fan speed
    } else {
        // Linear mapping: 6A->15%, 32A->100%
        fanSpeed = 15 + (current-6)*(100-15)/(32-6)
    }

    command := fmt.Sprintf("ascset|0,fan-spd,%d", fanSpeed)
    
    _, err := c.sendCommand(command)
    if err != nil {
        return fmt.Errorf("failed to set fan speed: %v", err)
    }

    return nil
}

// Additional mining-specific methods

// GetHashrate returns current hashrate in MH/s
func (c *AvalonQ) GetHashrate() (float64, error) {
    response, err := c.sendCommand("summary")
    if err != nil {
        return 0, err
    }

    data := c.parseResponse(response)
    
    if mhsAv, exists := data["MHS av"]; exists {
        return strconv.ParseFloat(mhsAv, 64)
    }

    return 0, fmt.Errorf("hashrate not found in response")
}

// GetTemperature returns average temperature
func (c *AvalonQ) GetTemperature() (float64, error) {
    response, err := c.sendCommand("estats")
    if err != nil {
        return 0, err
    }

    data := c.parseResponse(response)
    
    if tAvg, exists := data["TAvg"]; exists {
        return strconv.ParseFloat(tAvg, 64)
    }

    return 0, fmt.Errorf("temperature not found in response")
}

// GetVersion returns miner version info
func (c *AvalonQ) GetVersion() (string, error) {
    response, err := c.sendCommand("version")
    if err != nil {
        return "", err
    }

    data := c.parseResponse(response)
    
    if version, exists := data["CGMiner"]; exists {
        if model, exists := data["MODEL"]; exists {
            return fmt.Sprintf("%s %s", model, version), nil
        }
        return version, nil
    }

    return "", fmt.Errorf("version not found in response")
}

// SetWorkMode sets the work mode (0-2)
func (c *AvalonQ) SetWorkMode(mode int) error {
    if mode < 0 || mode > 2 {
        return fmt.Errorf("invalid work mode: %d (must be 0-2)", mode)
    }

    command := fmt.Sprintf("ascset|0,workmode,set,%d", mode)
    
    _, err := c.sendCommand(command)
    if err != nil {
        return fmt.Errorf("failed to set work mode: %v", err)
    }

    return nil
}

// SetPool configures mining pool
func (c *AvalonQ) SetPool(poolNum int, poolAddr, worker, workerPass string) error {
    if poolNum < 0 || poolNum > 2 {
        return fmt.Errorf("invalid pool number: %d (must be 0-2)", poolNum)
    }

    command := fmt.Sprintf("setpool|%s,%s,%d,%s,%s,%s", 
        c.username, c.password, poolNum, poolAddr, worker, workerPass)
    
    _, err := c.sendCommand(command)
    if err != nil {
        return fmt.Errorf("failed to set pool: %v", err)
    }

    return nil
}

// Reboot reboots the miner
func (c *AvalonQ) Reboot() error {
    _, err := c.sendCommand("ascset|0,reboot,0")
    if err != nil {
        return fmt.Errorf("failed to reboot miner: %v", err)
    }
    return nil
}
