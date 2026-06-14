package config

import mqxconfig "github.com/gospacex/mqx/config"

// TracingConfig is a Go type alias to mqx/config.TracingConfig. The alias
// is identity-preserving: field access, json tags, yaml tags, and method
// promotion all flow from the mqx type without conversion. Drift between
// vectorx and mqx is impossible because there is only one struct.
type TracingConfig = mqxconfig.TracingConfig
