// Package mqtt provides an MQTT client with auto-connect and reconnect (Eclipse Paho autopaho).
package mqtt

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// DefaultKeepAlive is the default MQTT keepalive period in seconds (max per spec; effectively keep forever).
const DefaultKeepAlive = 65535

// DefaultConnectTimeout is the default connection timeout.
const DefaultConnectTimeout = 30 * time.Second

// DefaultPort is the default MQTT port when Port is 0.
const DefaultPort = 1883

// Options configures the MQTT client. Endpoint is required to enable MQTT.
type Options struct {
	// Endpoint is the broker host (domain or IP only, e.g. "broker.example.com", "192.168.1.1").
	// Empty means MQTT disabled.
	Endpoint string
	// Port is the broker port (e.g. 1883, 8883). 0 uses DefaultPort.
	Port int
	// ClientID is the MQTT client identifier. If empty, a default is generated.
	ClientID string
	// Username and Password are optional broker credentials.
	Username string
	Password string
	// KeepAlive is the keepalive period in seconds (default DefaultKeepAlive; 0 = use default).
	KeepAlive uint16
	// ConnectTimeout is how long to wait for the initial connection (default DefaultConnectTimeout; 0 = use default).
	ConnectTimeout time.Duration
}

// ServerURL returns the broker URL as *url.URL for Paho (always mqtt://host:port).
// Validate must have been called first. Endpoint is domain or IP only.
func (o *Options) ServerURL() (*url.URL, error) {
	host := strings.TrimSpace(o.Endpoint)
	if host == "" {
		return nil, fmt.Errorf("mqtt: endpoint is required")
	}
	port := o.Port
	if port == 0 {
		port = DefaultPort
	}
	u := &url.URL{
		Scheme: "mqtt",
		Host:   fmt.Sprintf("%s:%d", host, port),
	}
	return u, nil
}
