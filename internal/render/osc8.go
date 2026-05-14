package render

func osc8Link(text, url string, enabled bool) string {
	if !enabled {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
