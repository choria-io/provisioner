package config

// Pause implements backplane.Pausable
func (c *Config) Pause() {
	c.Lock()
	defer c.Unlock()

	c.paused = true
	c.setPauseStat()
}

// Resume implements backplane.Pausable
func (c *Config) Resume() {
	c.Lock()
	defer c.Unlock()

	c.paused = false
	c.setPauseStat()
}

// Flip implements backplane.Pausable
func (c *Config) Flip() {
	c.Lock()
	defer c.Unlock()

	c.paused = !c.paused
	c.setPauseStat()
}

// Paused implements backplane.Pausable
func (c *Config) Paused() bool {
	c.Lock()
	defer c.Unlock()

	return c.paused
}

func (c *Config) setPauseStat() {
	if c.paused {
		pausedGauge.WithLabelValues(c.Site).Set(1)
	} else {
		pausedGauge.WithLabelValues(c.Site).Set(0)
	}
}
