package ghapp

// Enable all KMS providers in the default build. Each provider self-registers
// via its init() when compiled in, so published images support aws, azure,
// gcp, and vault out of the box without extra Go build tags. Individual
// providers can still be excluded at build time with opt-out tags
// (ghait.no_aws, ghait.no_azure, ghait.no_gcp, ghait.no_vault); each
// provider package ships a disabled.go stub for that tag, so these imports
// keep compiling to an empty package rather than failing the build.
//
// The file provider needs no import here: ghait registers it by default
// unless built with ghait.no_file.
import (
	_ "github.com/isometry/ghait/v84/provider/aws"
	_ "github.com/isometry/ghait/v84/provider/azure"
	_ "github.com/isometry/ghait/v84/provider/gcp"
	_ "github.com/isometry/ghait/v84/provider/vault"
)
