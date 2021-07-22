package priority

import (
	"testing"
	"time"
)

func TestFromTimeInterval(t *testing.T) {
	type args struct {
		from time.Time
		to   time.Time
	}
	tests := []struct {
		name string
		args args
		want Priority
	}{
		{
			name: "5 days",
			args: args{
				from: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				to:   time.Date(2021, 1, 6, 0, 0, 0, 0, time.UTC),
			},
			want: 5,
		},
		{
			name: "30 days",
			args: args{
				from: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				to:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Add(30 * 24 * time.Hour),
			},
			want: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromTimeInterval(tt.args.from, tt.args.to); got != tt.want {
				t.Errorf("FromTimeInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_LowerBy(t *testing.T) {
	type args struct {
		val int
	}
	tests := []struct {
		name string
		p    Priority
		args args
		want Priority
	}{
		{
			name: "lower by 4",
			p:    8,
			args: args{val: 4},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.LowerBy(tt.args.val); got != tt.want {
				t.Errorf("LowerBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_RaiseBy(t *testing.T) {
	type args struct {
		val int
	}
	tests := []struct {
		name string
		p    Priority
		args args
		want Priority
	}{
		{
			name: "raise by 4",
			p:    5,
			args: args{val: 4},
			want: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.RaiseBy(tt.args.val); got != tt.want {
				t.Errorf("RaiseBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_Lower(t *testing.T) {
	tests := []struct {
		name string
		p    Priority
		want Priority
	}{
		{
			name: "testing lower",
			p:    4,
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Lower(); got != tt.want {
				t.Errorf("Lower() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_Raise(t *testing.T) {
	tests := []struct {
		name string
		p    Priority
		want Priority
	}{
		{
			name: "testing raise",
			p:    5,
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Raise(); got != tt.want {
				t.Errorf("Raise() = %v, want %v", got, tt.want)
			}
		})
	}
}
