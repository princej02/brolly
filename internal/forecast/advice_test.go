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
		{
			name: "umbrella only: rainy, but warm and calm",
			hours: []Hour{
				h(0, 20, 60, 24),
				h(1, 18, 70, 20),
				h(2, 16, 55, 10),
			},
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
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
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
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
		{
			name: "both jackets: two triggers (rainy, cold)",
			hours: []Hour{
				h(0, 10, 60, 15),
				h(1, 9, 65, 10),
				h(2, 11, 40, 5),
			},
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 60},
					{Rule: RuleJacketCold, Threshold: 12, Observed: 10},
				},
				Rainiest: Peak[int]{At: at(1), Value: 65},
				Windiest: Peak[float64]{At: at(0), Value: 15},
				Coldest:  Peak[float64]{At: at(1), Value: 9},
			},
			wantErr: nil,
		},
		{
			name: "neither: no triggers",
			hours: []Hour{
				h(0, 22, 10, 15),
				h(1, 25, 20, 20),
				h(2, 21, 5, 10),
			},
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
			want: Advice{
				Triggers: nil,
				Rainiest: Peak[int]{At: at(1), Value: 20},
				Windiest: Peak[float64]{At: at(1), Value: 20},
				Coldest:  Peak[float64]{At: at(2), Value: 21},
			},
			wantErr: nil,
		},
		{
			name: "tied rainiest: two hours share highest rain chance",
			hours: []Hour{
				h(0, 20, 70, 0),
				h(1, 20, 70, 0),
			},
			rules: Rules{UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30},
			want: Advice{
				Rainiest: Peak[int]{Value: 70, At: h(0, 0, 0, 0).At},
				Coldest:  Peak[float64]{Value: 20, At: h(0, 0, 0, 0).At},
				Windiest: Peak[float64]{Value: 0, At: h(0, 0, 0, 0).At},
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 70},
				},
			},
			wantErr: nil,
		},
		{
			name: "tied coldest: two hours share the lowest temperature.",
			hours: []Hour{
				h(0, 10, 10, 0),
				h(1, 10, 20, 0),
			},
			rules: Rules{UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30},
			want: Advice{
				Rainiest: Peak[int]{Value: 20, At: h(1, 0, 0, 0).At},
				Coldest:  Peak[float64]{Value: 10, At: h(0, 0, 0, 0).At},
				Windiest: Peak[float64]{Value: 0, At: h(0, 0, 0, 0).At},
				Triggers: []Trigger{
					{Rule: RuleJacketCold, Threshold: 12, Observed: 10},
				},
			},
			wantErr: nil,
		},
		{
			name: "tied windiest: two hours share highest wind",
			hours: []Hour{
				h(0, 20, 10, 35),
				h(1, 20, 20, 35),
			},
			rules: Rules{UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30},
			want: Advice{
				Rainiest: Peak[int]{Value: 20, At: h(1, 0, 0, 0).At},
				Coldest:  Peak[float64]{Value: 20, At: h(0, 0, 0, 0).At},
				Windiest: Peak[float64]{Value: 35, At: h(0, 0, 0, 0).At},
				Triggers: []Trigger{
					{Rule: RuleJacketWind, Threshold: 30, Observed: 35},
				},
			},
			wantErr: nil,
		},
		{
			name: "each boundary",
			hours: []Hour{
				h(0, 12, 50, 30),
			},
			rules: Rules{
				UmbrellaAtPct: 50, JacketBelowC: 12, JacketWindKmh: 30,
			},
			want: Advice{
				Triggers: []Trigger{
					{Rule: RuleUmbrella, Threshold: 50, Observed: 50},
					{Rule: RuleJacketCold, Threshold: 12, Observed: 12},
					{Rule: RuleJacketWind, Threshold: 30, Observed: 30},
				},
				Rainiest: Peak[int]{At: at(0), Value: 50},
				Windiest: Peak[float64]{At: at(0), Value: 30},
				Coldest:  Peak[float64]{At: at(0), Value: 12},
			},
			wantErr: nil,
		},
		{
			name: "an invalid rule",
			hours: []Hour{
				h(0, 12, 50, 30),
			},
			rules: Rules{
				UmbrellaAtPct: -1, JacketBelowC: 12, JacketWindKmh: 30,
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
