package cli

import "testing"

func TestShouldColor(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		stdoutTTY bool
		want      bool
	}{
		{"NO_COLOR truthy + TTY => false", map[string]string{"NO_COLOR": "1"}, true, false},
		{"NO_COLOR empty + TTY => true (falls through)", map[string]string{"NO_COLOR": ""}, true, true},
		{"FORCE_COLOR=1 + no TTY => true", map[string]string{"FORCE_COLOR": "1"}, false, true},
		{"FORCE_COLOR=0 + TTY => false (BUG FIX pinned)", map[string]string{"FORCE_COLOR": "0"}, true, false},
		{"empty env + no TTY => false", map[string]string{}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldColor(tc.env, tc.stdoutTTY)
			if got != tc.want {
				t.Fatalf("ShouldColor(%v, %v) = %v, want %v", tc.env, tc.stdoutTTY, got, tc.want)
			}
		})
	}
}

func TestShouldOSC8(t *testing.T) {
	if !ShouldOSC8(true) {
		t.Fatalf("ShouldOSC8(true) = false, want true")
	}
	if ShouldOSC8(false) {
		t.Fatalf("ShouldOSC8(false) = true, want false")
	}
}
