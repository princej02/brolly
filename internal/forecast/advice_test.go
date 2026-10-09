package forecast

import (
	"errors"
	"testing"
	"time"
)

var base = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

func h(hour int, tempC float64, rainPct int, windKmh float64) Hour {
	return Hour{
		At:            base.Add(time.Duration(hour) * time.Hour),
		TempC:         tempC,
		RainChancePct: rainPct,
		WindKmh:       windKmh,
	}
}

func at(hour int) time.Time {
	return base.Add(time.Duration(hour) * time.Hour)
}

func peakEq[T comparable](a, b Peak[T]) bool {
	return a.Value == b.Value && a.At.Equal(b.At)
}

func triggersEq(a, b []Trigger) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func adviceEq(a, b Advice) bool {
	return triggersEq(a.Triggers, b.Triggers) &&
		peakEq(a.Rainiest, b.Rainiest) &&
		peakEq(a.Windiest, b.Windiest) &&
		peakEq(a.Coldest, b.Coldest)
}

func TestAdvise(t *testing.T) {
	testCases := []struct {
		name    string
		hours   []Hour
		rules   Rules
		want    Advice
		wantErr error
	}{
		// Empty input
		{
			name:    "empty hours: returns empty advice and ErrNoHours",
			hours:   []Hour{},
			rules:   DefaultRules(),
			want:    Advice{},
			wantErr: ErrNoHours,
		},
		// Standard trigger scenarios
		{
			name: "umbrella only: rainy, but warm and calm",
			hours: []Hour{
				h(0, 20, 60, 24),
				h(1, 18, 70, 20),
				h(2, 16, 55, 10),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 60},
				},
				Rainiest: Peak[int]{At: at(1), Value: 70},
				Windiest: Peak[float64]{At: at(0), Value: 24},
				Coldest:  Peak[float64]{At: at(2), Value: 16},
			},
			wantErr: nil,
		},
		{
			name: "cold only: dry and calm, but cold",
			hours: []Hour{
				h(0, 15, 10, 10),
				h(1, 8, 20, 15),
				h(2, 11, 0, 5),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleJacketCold, Threshold: 12, Observed: 8},
				},
				Rainiest: Peak[int]{At: at(1), Value: 20},
				Windiest: Peak[float64]{At: at(1), Value: 15},
				Coldest:  Peak[float64]{At: at(1), Value: 8},
			},
			wantErr: nil,
		},
		{
			name: "wind only: dry and warm but windy",
			hours: []Hour{
				h(0, 20, 10, 25),
				h(1, 18, 15, 35),
				h(2, 19, 5, 20),
			},
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleJacketWind, Threshold: 30, Observed: 35},
				},
				Rainiest: Peak[int]{At: at(1), Value: 15},
				Windiest: Peak[float64]{At: at(1), Value: 35},
				Coldest:  Peak[float64]{At: at(1), Value: 18},
			},
			wantErr: nil,
		},
		// Both jacket triggers & All triggers combined
		{
			name: "both jackets: cold and windy, but dry",
			hours: []Hour{
				h(0, 10, 20, 35),
				h(1, 15, 10, 15),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleJacketCold, Threshold: 12, Observed: 10},
					{Rule: RuleJacketWind, Threshold: 30, Observed: 35},
				},
				Rainiest: Peak[int]{At: at(0), Value: 20},
				Windiest: Peak[float64]{At: at(0), Value: 35},
				Coldest:  Peak[float64]{At: at(0), Value: 10},
			},
			wantErr: nil,
		},
		{
			name: "everything: rainy, cold, and windy (pins trigger order)",
			hours: []Hour{
				h(0, 9, 65, 35),
				h(1, 14, 20, 10),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 65},
					{Rule: RuleJacketCold, Threshold: 12, Observed: 9},
					{Rule: RuleJacketWind, Threshold: 30, Observed: 35},
				},
				Rainiest: Peak[int]{At: at(0), Value: 65},
				Windiest: Peak[float64]{At: at(0), Value: 35},
				Coldest:  Peak[float64]{At: at(0), Value: 9},
			},
			wantErr: nil,
		},
		{
			name: "neither: benign conditions, no triggers",
			hours: []Hour{
				h(0, 22, 10, 15),
				h(1, 25, 20, 20),
				h(2, 21, 5, 10),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: nil,
				Rainiest: Peak[int]{At: at(1), Value: 20},
				Windiest: Peak[float64]{At: at(1), Value: 20},
				Coldest:  Peak[float64]{At: at(2), Value: 21},
			},
			wantErr: nil,
		},
		// Boundary precision pairs (pining >= vs <)
		{
			name:  "boundary: umbrella exact threshold triggers (50%)",
			hours: []Hour{h(0, 20, 50, 10)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{{Rule: RuleUmbrella, Threshold: 50, Observed: 50}},
				Rainiest: Peak[int]{At: at(0), Value: 50},
				Windiest: Peak[float64]{At: at(0), Value: 10},
				Coldest:  Peak[float64]{At: at(0), Value: 20},
			},
			wantErr: nil,
		},
		{
			name:  "boundary: umbrella just below threshold does not trigger (49%)",
			hours: []Hour{h(0, 20, 49, 10)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: nil,
				Rainiest: Peak[int]{At: at(0), Value: 49},
				Windiest: Peak[float64]{At: at(0), Value: 10},
				Coldest:  Peak[float64]{At: at(0), Value: 20},
			},
			wantErr: nil,
		},
		{
			name:  "boundary: cold at exact threshold does not trigger (12.0C is not below 12C)",
			hours: []Hour{h(0, 12.0, 10, 10)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: nil,
				Rainiest: Peak[int]{At: at(0), Value: 10},
				Windiest: Peak[float64]{At: at(0), Value: 10},
				Coldest:  Peak[float64]{At: at(0), Value: 12.0},
			},
			wantErr: nil,
		},
		{
			name:  "boundary: cold just below threshold triggers (11.9C)",
			hours: []Hour{h(0, 11.9, 10, 10)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{{Rule: RuleJacketCold, Threshold: 12, Observed: 11.9}},
				Rainiest: Peak[int]{At: at(0), Value: 10},
				Windiest: Peak[float64]{At: at(0), Value: 10},
				Coldest:  Peak[float64]{At: at(0), Value: 11.9},
			},
			wantErr: nil,
		},
		{
			name:  "boundary: wind exact threshold triggers (30.0 km/h)",
			hours: []Hour{h(0, 20, 10, 30.0)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{{Rule: RuleJacketWind, Threshold: 30, Observed: 30.0}},
				Rainiest: Peak[int]{At: at(0), Value: 10},
				Windiest: Peak[float64]{At: at(0), Value: 30.0},
				Coldest:  Peak[float64]{At: at(0), Value: 20},
			},
			wantErr: nil,
		},
		{
			name:  "boundary: wind just below threshold does not trigger (29.9 km/h)",
			hours: []Hour{h(0, 20, 10, 29.9)},
			rules: DefaultRules(),
			want: Advice{
				Triggers: nil,
				Rainiest: Peak[int]{At: at(0), Value: 10},
				Windiest: Peak[float64]{At: at(0), Value: 29.9},
				Coldest:  Peak[float64]{At: at(0), Value: 20},
			},
			wantErr: nil,
		},
		// Decoupled ties: non-tested dimensions strictly differ
		{
			name: "tied rainiest: prefers earlier hour (temp and wind differ)",
			hours: []Hour{
				h(0, 22, 70, 10),
				h(1, 18, 70, 25),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 70},
				},
				Rainiest: Peak[int]{At: at(0), Value: 70},
				Coldest:  Peak[float64]{At: at(1), Value: 18},
				Windiest: Peak[float64]{At: at(1), Value: 25},
			},
			wantErr: nil,
		},
		{
			name: "tied coldest: prefers earlier hour (rain and wind differ)",
			hours: []Hour{
				h(0, 10, 15, 12),
				h(1, 10, 25, 18),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleJacketCold, Threshold: 12, Observed: 10},
				},
				Rainiest: Peak[int]{At: at(1), Value: 25},
				Coldest:  Peak[float64]{At: at(0), Value: 10},
				Windiest: Peak[float64]{At: at(1), Value: 18},
			},
			wantErr: nil,
		},
		{
			name: "tied windiest: prefers earlier hour (temp and rain differ)",
			hours: []Hour{
				h(0, 22, 10, 35),
				h(1, 17, 20, 35),
			},
			rules: DefaultRules(),
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleJacketWind, Threshold: 30, Observed: 35},
				},
				Rainiest: Peak[int]{At: at(1), Value: 20},
				Coldest:  Peak[float64]{At: at(1), Value: 17},
				Windiest: Peak[float64]{At: at(0), Value: 35},
			},
			wantErr: nil,
		},
		// Invalid rule permutations
		{
			name:  "invalid rule: negative rain percentage",
			hours: []Hour{h(0, 15, 20, 10)},
			rules: Rules{
				UmbrellaAtPct: -1,
				JacketBelowC:  12,
				JacketWindKmh: 30,
			},
			want:    Advice{},
			wantErr: ErrInvalidRule,
		},
		{
			name:  "invalid rule: rain percentage exceeds 100",
			hours: []Hour{h(0, 15, 20, 10)},
			rules: Rules{
				UmbrellaAtPct: 101,
				JacketBelowC:  12,
				JacketWindKmh: 30,
			},
			want:    Advice{},
			wantErr: ErrInvalidRule,
		},
		{
			name:  "invalid rule: negative wind speed threshold",
			hours: []Hour{h(0, 15, 20, 10)},
			rules: Rules{
				UmbrellaAtPct: 50,
				JacketBelowC:  12,
				JacketWindKmh: -5,
			},
			want:    Advice{},
			wantErr: ErrInvalidRule,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			advise, err := Advise(tc.hours, tc.rules)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error: %v, got: %v", tc.wantErr, err)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error: %v, got: %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if !adviceEq(advise, tc.want) {
				t.Errorf("expected advice: %v, got: %v", tc.want, advise)
			}
		})
	}
}
