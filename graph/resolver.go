package graph

import (
	"git.blinkpay.vn/software/authix/config"
	"git.blinkpay.vn/software/authix/internal/persistence"
)

// import (
//
//	keto "github.com/ory/keto-client-go/v25"
//
// )
//
// //go:generate go tool gqlgen generate
type Resolver struct {
	PostgresPersister persistence.Persister
	// KetoClient        *keto.APIClient
	Config config.Config
}
