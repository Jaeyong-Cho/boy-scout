package cohesion

import (
	"testing"
)

func TestCompute_CallEdgeConnectsLCOM4ButNotTCC(t *testing.T) {
	// Given: setX (touches x), setY (touches y), reset (calls both, touches nothing)
	// LCOM4 should be 1 (all connected via calls)
	// TCC should be 0 (setX/setY don't share a field)
	// LCC should be 1.0 (all reachable via reset)
	methods := []Method{
		{
			Name:   "setX",
			Fields: map[string]bool{"x": true},
			Calls:  map[string]bool{},
		},
		{
			Name:   "setY",
			Fields: map[string]bool{"y": true},
			Calls:  map[string]bool{},
		},
		{
			Name:   "reset",
			Fields: map[string]bool{},
			Calls:  map[string]bool{"setX": true, "setY": true},
		},
	}

	score := Compute(methods)

	if score.LCOM4 != 1 {
		t.Errorf("LCOM4: got %d, want 1", score.LCOM4)
	}
	if score.TCC != 0.0 { // 0/3 pairs = 0
		t.Errorf("TCC: got %f, want 0.0", score.TCC)
	}
	if score.LCC != 1.0 { // all connected via reset: 3/3 pairs = 1.0
		t.Errorf("LCC: got %f, want 1.0", score.LCC)
	}
	if score.LCOM4Level != "good" {
		t.Errorf("LCOM4Level: got %q, want %q", score.LCOM4Level, "good")
	}
	if score.TCCLevel != "danger" {
		t.Errorf("TCCLevel: got %q, want %q", score.TCCLevel, "danger")
	}
	if score.LCCLevel != "good" {
		t.Errorf("LCCLevel: got %q, want %q", score.LCCLevel, "good")
	}
}

func TestCompute_TwoUnrelatedMethods(t *testing.T) {
	// Given: two methods that share no field and don't call each other
	methods := []Method{
		{Name: "methodA", Fields: map[string]bool{}, Calls: map[string]bool{}},
		{Name: "methodB", Fields: map[string]bool{}, Calls: map[string]bool{}},
	}

	score := Compute(methods)

	if score.LCOM4 != 2 {
		t.Errorf("LCOM4: got %d, want 2", score.LCOM4)
	}
	if score.TCC != 0.0 {
		t.Errorf("TCC: got %f, want 0.0", score.TCC)
	}
	if score.LCC != 0.0 {
		t.Errorf("LCC: got %f, want 0.0", score.LCC)
	}
}

func TestLCOM4Level_Boundaries(t *testing.T) {
	tests := []struct {
		lcom4 int
		want  string
	}{
		{1, "good"},
		{2, "warning"},
		{3, "danger"},
		{4, "danger"},
	}
	for _, tt := range tests {
		// Create exactly tt.lcom4 disconnected components
		// Each component gets 2 methods (minimum to be valid for Compute)
		// They share a field within each component, but no field shared across components
		numMethods := tt.lcom4 * 2
		methods := make([]Method, numMethods)
		for i := 0; i < tt.lcom4; i++ {
			// Component i: two methods sharing field_i
			methods[2*i] = Method{
				Name:   string(rune(65 + 2*i)),
				Fields: map[string]bool{string(rune(97 + i)): true}, // field_a, field_b, etc.
				Calls:  map[string]bool{},
			}
			methods[2*i+1] = Method{
				Name:   string(rune(65 + 2*i + 1)),
				Fields: map[string]bool{string(rune(97 + i)): true},
				Calls:  map[string]bool{},
			}
		}
		score := Compute(methods)
		if score.LCOM4 != tt.lcom4 {
			t.Errorf("LCOM4=%d: got LCOM4=%d", tt.lcom4, score.LCOM4)
		}
		if score.LCOM4Level != tt.want {
			t.Errorf("LCOM4=%d: got %q, want %q", tt.lcom4, score.LCOM4Level, tt.want)
		}
	}
}

func TestRatioLevel_Boundaries(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		want string
	}{
		{"just_above_0.8", 0.81, "good"},
		{"exactly_0.8", 0.8, "warning"},
		{"mid_range", 0.65, "warning"},
		{"exactly_0.5", 0.5, "warning"},
		{"just_below_0.5", 0.49, "danger"},
		{"zero", 0.0, "danger"},
	}
	for _, tt := range tests {
		got := ratioLevel(tt.val)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCompute_PanicsUnderTwoMethods(t *testing.T) {
	// Should panic when called with fewer than 2 methods
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Compute with 1 method: expected panic, got none")
		}
	}()
	Compute([]Method{{Name: "a", Fields: map[string]bool{}, Calls: map[string]bool{}}})
}

func TestWorst_PicksMostSevere(t *testing.T) {
	tests := []struct {
		name string
		s    Score
		want string
	}{
		{
			name: "all_good",
			s:    Score{LCOM4Level: "good", TCCLevel: "good", LCCLevel: "good"},
			want: "good",
		},
		{
			name: "one_warning",
			s:    Score{LCOM4Level: "good", TCCLevel: "warning", LCCLevel: "good"},
			want: "warning",
		},
		{
			name: "one_danger",
			s:    Score{LCOM4Level: "good", TCCLevel: "good", LCCLevel: "danger"},
			want: "danger",
		},
		{
			name: "danger_beats_warning",
			s:    Score{LCOM4Level: "warning", TCCLevel: "danger", LCCLevel: "warning"},
			want: "danger",
		},
	}
	for _, tt := range tests {
		got := Worst(tt.s)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
