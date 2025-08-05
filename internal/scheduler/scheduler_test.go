package scheduler

import (
	"testing"
	"time"
)

func TestCalculateNextRun(t *testing.T) {
	s := &service{}
	
	tests := []struct {
		name     string
		interval string
		from     time.Time
		wantMin  time.Duration // minimum expected duration
		wantMax  time.Duration // maximum expected duration (accounting for jitter)
	}{
		{
			name:     "1 hour interval",
			interval: "1h",
			from:     time.Now(),
			wantMin:  1 * time.Hour,
			wantMax:  1*time.Hour + 5*time.Minute, // 1h + up to 5m jitter
		},
		{
			name:     "1 minute interval",
			interval: "1m",
			from:     time.Now(),
			wantMin:  1 * time.Minute,
			wantMax:  1*time.Minute + 5*time.Minute, // 1m + up to 5m jitter
		},
		{
			name:     "60 seconds interval",
			interval: "60s",
			from:     time.Now(),
			wantMin:  60 * time.Second,
			wantMax:  60*time.Second + 5*time.Minute, // 60s + up to 5m jitter
		},
		{
			name:     "3600 seconds interval",
			interval: "3600s",
			from:     time.Now(),
			wantMin:  3600 * time.Second,
			wantMax:  3600*time.Second + 5*time.Minute, // 3600s + up to 5m jitter
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.calculateNextRun(tt.interval, tt.from)
			if got.IsZero() {
				t.Errorf("calculateNextRun() returned zero time")
				return
			}
			
			duration := got.Sub(tt.from)
			if duration < tt.wantMin {
				t.Errorf("calculateNextRun() duration = %v, want at least %v", duration, tt.wantMin)
			}
			if duration > tt.wantMax {
				t.Errorf("calculateNextRun() duration = %v, want at most %v", duration, tt.wantMax)
			}
			
			// Log the actual values for debugging
			t.Logf("Interval: %s, Duration: %v, NextRun: %v", tt.interval, duration, got)
		})
	}
}

func TestIsValidScheduleInterval(t *testing.T) {
	s := &service{}
	
	tests := []struct {
		name     string
		interval string
		want     bool
	}{
		{
			name:     "valid duration - 1h",
			interval: "1h",
			want:     true,
		},
		{
			name:     "valid duration - 60s",
			interval: "60s",
			want:     true,
		},
		{
			name:     "valid duration - 1m",
			interval: "1m",
			want:     true,
		},
		{
			name:     "valid exact time",
			interval: "exact:14:00",
			want:     true,
		},
		{
			name:     "valid exact multiple times",
			interval: "exact:09:00,14:00,20:00",
			want:     true,
		},
		{
			name:     "invalid format",
			interval: "invalid",
			want:     false,
		},
		{
			name:     "empty string",
			interval: "",
			want:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.isValidScheduleInterval(tt.interval)
			if got != tt.want {
				t.Errorf("isValidScheduleInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}