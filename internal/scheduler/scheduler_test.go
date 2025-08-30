package scheduler

import (
	"os"
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

func TestCalculateNextRunWithTimezones(t *testing.T) {
	s := &service{}
	
	// Test the critical bug: ensure negative time differences never happen
	// This simulates the exact scenario where TZ=US/New_York causes every-minute execution
	tests := []struct {
		name     string
		timezone string
		interval string
	}{
		{
			name:     "America/New_York with exact time",
			timezone: "America/New_York",
			interval: "exact:14:00",
		},
		{
			name:     "America/Chicago with exact time", 
			timezone: "America/Chicago",
			interval: "exact:14:00",
		},
		{
			name:     "Europe/London with exact time",
			timezone: "Europe/London", 
			interval: "exact:14:00",
		},
	}
	
	// Save original TZ
	originalTZ := os.Getenv("TZ")
	defer func() {
		if originalTZ != "" {
			os.Setenv("TZ", originalTZ)  
		} else {
			os.Unsetenv("TZ")
		}
	}()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the TZ environment variable (this is what causes the bug)
			os.Setenv("TZ", tt.timezone)
			
			// Test at different times of day
			testTimes := []struct {
				hour int
				name string
			}{
				{10, "morning"},
				{14, "exactly_at_target"},
				{16, "afternoon"},
				{20, "evening"},
			}
			
			for _, testTime := range testTimes {
				// Create a test time
				now := time.Now()
				from := time.Date(now.Year(), now.Month(), now.Day(), testTime.hour, 30, 0, 0, time.Local)
				
				// Calculate next run
				nextRun := s.calculateNextRun(tt.interval, from)
				
				if nextRun.IsZero() {
					t.Errorf("calculateNextRun() returned zero time for %s", testTime.name)
					continue
				}
				
				// The critical check: ensure time difference is never negative
				diff := nextRun.Sub(from)
				if diff < 0 {
					t.Errorf("CRITICAL BUG: Negative time difference %v at %s would cause every-minute execution!", 
						diff, testTime.name)
					t.Errorf("From: %v, NextRun: %v", from, nextRun)
				}
				
				// Ensure minimum time is at least 1 minute (to avoid every-minute execution)
				if diff < time.Minute {
					t.Errorf("Time difference too small (%v) at %s, could cause frequent execution", 
						diff, testTime.name)
				}
				
				t.Logf("TZ=%s, %s: From=%v, NextRun=%v, Diff=%v", 
					tt.timezone, testTime.name, from, nextRun, diff)
			}
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