package config

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"

	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/rs/zerolog/log"
)

func writeOutMetadataInfo(mACL presentation.MetadataACL, metadata sync.MetadataConfig) {
	for mt := range metadata.Fields {
		role, ok := mACL[mt]
		if !ok {
			role = dbo.RoleGuest
		}
		log.Logger.Info().
			Str("metadata", mt).
			Str("minimum_role", string(role)).Msg("Set minimum visibility")
	}
}
