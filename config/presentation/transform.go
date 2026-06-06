package presentation

func (pc *PresentationConfig) TransformAfterValidation() error {
	pc.ConvertedMetadataACL = MetadataACL{}
	for role, list := range pc.MetadataACL {
		for _, metadata := range list {
			pc.ConvertedMetadataACL[metadata] = role
		}
	}
	for key := range pc.TagMeaningConfig.MeaningMap {
		meaning := pc.TagMeaningConfig.MeaningMap[key]
		meaning.FeaturesMap = make(map[TagFeature]struct{})
		for _, feature := range meaning.Features {
			meaning.FeaturesMap[feature] = struct{}{}
		}
		pc.TagMeaningConfig.MeaningMap[key] = meaning
	}
	return nil
}
