package shared

import "testing"

func TestParsePercent(t *testing.T) {
	tests := []struct {
		s       string
		want    int
		wantErr bool
	}{
		{s: "", wantErr: true},
		{s: "-1", wantErr: true},
		{s: "-4", wantErr: true},
		{s: "300", wantErr: true},
		{s: "0", want: 0},
		{s: "50", want: 50},
		{s: "100", want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got, err := parsePercent(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePercent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePercent() = %v, want %v", got, tt.want)
			}
		})
	}
}
