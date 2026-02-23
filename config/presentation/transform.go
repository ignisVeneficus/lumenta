package presentation

func (pc *PresentationConfig) TransformAfterValidation() error {
	pc.ConvertedMetadataACL = MetadataACL{}
	for role, list := range pc.MetadataACL {
		for _, metadata := range list {
			pc.ConvertedMetadataACL[metadata] = role
		}
	}
	return nil
}
