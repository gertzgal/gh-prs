package cli

import "strings"

func truthyFlag(v string, present bool) bool {
	if !present || v == "" || v == "0" || strings.EqualFold(v, "false") {
		return false
	}
	return true
}

// ShouldColor decides whether ANSI color escapes are emitted.
// Rules (in order):
//  1. NO_COLOR truthy => false
//  2. FORCE_COLOR set => parse FORCE_COLOR as truthy-flag
//  3. fall back to stdoutIsTTY
func ShouldColor(env map[string]string, stdoutIsTTY bool) bool {
	if v, ok := env["NO_COLOR"]; ok && truthyFlag(v, ok) {
		return false
	}
	if v, ok := env["FORCE_COLOR"]; ok {
		return truthyFlag(v, ok)
	}
	return stdoutIsTTY
}

// ShouldOSC8 returns stdoutIsTTY. NO_COLOR does not gate OSC8.
func ShouldOSC8(stdoutIsTTY bool) bool { return stdoutIsTTY }
