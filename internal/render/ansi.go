package render

func sgr(code string, s string, on bool) string {
	if !on {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func fgGreen(s string, on bool) string        { return sgr("32", s, on) }
func fgRed(s string, on bool) string          { return sgr("31", s, on) }
func fgYellow(s string, on bool) string       { return sgr("33", s, on) }
func fgGray(s string, on bool) string         { return sgr("90", s, on) }
func fgBrightYellow(s string, on bool) string { return sgr("93", s, on) }
func styleBold(s string, on bool) string      { return sgr("1", s, on) }
func styleDim(s string, on bool) string       { return sgr("2", s, on) }

func osc8Link(text, url string, enabled bool) string {
	if !enabled {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
