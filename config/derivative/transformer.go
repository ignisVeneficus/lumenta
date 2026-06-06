package derivative

func (derivatives *DerivativesConfig) TransformAfterValidation() error {
	for i := range *derivatives {
		d := &(*derivatives)[i]
		if d.JPGQuality < 1 || d.JPGQuality > 100 {
			d.JPGQuality = 75
		}
	}
	return nil
}
