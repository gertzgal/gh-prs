package cli

// Flags mirrors the TS Args type. Cobra handles --help/-h; Help is kept here
// for symmetry with the TS surface.
type Flags struct {
	JSON  bool
	Debug bool
	Help  bool
}

// composeFlags merges cobra-parsed flags with env. DEBUG=<non-empty> in env
// enables debug even if --debug was not passed.
func composeFlags(cobraJSON, cobraDebug bool, env map[string]string) Flags {
	debug := cobraDebug
	if !debug {
		if v, ok := env["DEBUG"]; ok && v != "" {
			debug = true
		}
	}
	return Flags{JSON: cobraJSON, Debug: debug}
}
