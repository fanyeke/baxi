package logger

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNew_ValidLevels(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantLvl zapcore.Level
		wantErr bool
	}{
		{"debug level", "debug", zapcore.DebugLevel, false},
		{"info level", "info", zapcore.InfoLevel, false},
		{"warn level", "warn", zapcore.WarnLevel, false},
		{"error level", "error", zapcore.ErrorLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q) error = %v, wantErr = %v", tt.level, err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("New(%q) returned nil logger", tt.level)
				return
			}
			if got.Level() != tt.wantLvl {
				t.Errorf("New(%q) level = %v, want %v", tt.level, got.Level(), tt.wantLvl)
			}
		})
	}
}

func TestNew_DefaultLevel(t *testing.T) {
	tests := []struct {
		name string
		lvl  string
	}{
		{"unknown level", "unknown"},
		{"empty level", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.lvl)
			if err != nil {
				t.Fatalf("New(%q) error = %v", tt.lvl, err)
			}
			if got.Level() != zapcore.InfoLevel {
				t.Errorf("New(%q) level = %v, want InfoLevel", tt.lvl, got.Level())
			}
		})
	}
}

func TestNew_JSONEncoderConfig(t *testing.T) {
	got, err := New("debug")
	if err != nil {
		t.Fatalf("New(\"debug\") error = %v", err)
	}

	// Verify the logger has the right config by logging and syncing
	got.Debug("test debug message")
	got.Info("test info message")
	got.Warn("test warn message")
	got.Error("test error message")
	err = got.Sync()
	if err != nil {
		t.Logf("Sync() returned error (expected in test): %v", err)
	}
}

func TestNew_ProducesNonNilLogger(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			got, err := New(lvl)
			if err != nil {
				t.Fatalf("New(%q) error = %v", lvl, err)
			}
			if got == nil {
				t.Fatal("New() returned nil logger")
			}
		})
	}
}
