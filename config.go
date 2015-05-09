package zoom

// defaultConfiguration holds the default values for each config option
// if the zero value is provided in the input configuration, the value
// will fallback to the default value
var defaultConfiguration = Configuration{
	Address:  "localhost:6379",
	Network:  "tcp",
	Database: 0,
	Password: "",
}

// parseConfig returns a well-formed configuration struct.
// If the passedConfig is nil, returns defaultConfiguration.
// Else, for each zero value field in passedConfig,
// use the default value for that field.
func parseConfig(passedConfig *Configuration) *Configuration {
	if passedConfig == nil {
		return &defaultConfiguration
	}
	// copy the passedConfig
	newConfig := *passedConfig
	if newConfig.Address == "" {
		newConfig.Address = defaultConfiguration.Address
	}
	if newConfig.Network == "" {
		newConfig.Network = defaultConfiguration.Network
	}
	// since the zero value for int is 0, we can skip config.Database
	// since the zero value for string is "", we can skip config.Address
	return &newConfig
}

// Configuration contains various options. It should be created once
// and passed in to the Init function during application startup.
type Configuration struct {
	// Address to connect to. Default: "localhost:6379"
	Address string
	// Network to use. Default: "tcp"
	Network string
	// Database id to use (using SELECT). Default: 0
	Database int
	// Password for a password-protected redis database. If not empty,
	// every connection will use the AUTH command during initialization
	// to authenticate with the database. Default: ""
	Password string
}
