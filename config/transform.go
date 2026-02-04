package config

func (c *Config) TransformBeforeValidation() error {
	_ = c.Sync.TransformBeforeValidation()

	return nil
}

func (c *Config) TransformAfterValidation() error {
	_ = c.Sync.TransformAfterValidation()
	return nil
}
