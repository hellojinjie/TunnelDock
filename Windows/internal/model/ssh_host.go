package model

type HostAvailability string

const (
	HostAvailable          HostAvailability = "available"
	HostConfigurationError HostAvailability = "configuration-error"
	HostMissing            HostAvailability = "missing"
)

type SSHHost struct {
	Alias        string
	Hostname     string
	User         string
	Port         uint16
	ConfigOrder  int
	Availability HostAvailability
	Error        string
}
