package config

// FactData implements backplane.InfoSource
func (c *Config) FactData() interface{} {
	return c
}

// Version implements backplane.InfoSource
func (c *Config) Version() string {
	return Version
}
