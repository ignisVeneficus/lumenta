package config

func (c *Config) TransformBeforeValidation() error {
	_ = c.Sync.TransformBeforeValidation()

	return nil
}

func (c *Config) TransformAfterValidation() error {
	_ = c.Sync.TransformAfterValidation()
	_ = c.Auth.TransformAfterValidation()
	return nil
}
