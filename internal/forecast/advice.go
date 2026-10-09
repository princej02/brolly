package forecast

import (
	"errors"
	"fmt"
	"time"
)

type Hour struct {
	At            time.Time
	TempC         float64
	RainChancePct int
	WindKmh       float64
}

type RuleID int

const (
	RuleUnknown RuleID = iota
	RuleUmbrella
	RuleJacketCold
	RuleJacketWind
)

type Trigger struct {
	Rule      RuleID
	Threshold float64
	Observed  float64
}

type Rules struct {
	UmbrellaAtPct int
	JacketBelowC  float64
	JacketWindKmh float64
}

type Peak[T any] struct {
	Value T
	At    time.Time
}

type Advice struct {
	Triggers []Trigger

	Rainiest Peak[int]
	Coldest  Peak[float64]
	Windiest Peak[float64]
}

var (
	ErrNoHours     = errors.New("forecast: no hours available")
	ErrInvalidRule = errors.New("forecast: invalid rule")
)

func (r RuleID) String() string {
	switch r {
	case RuleUmbrella:
		return "Umbrella"
	case RuleJacketCold:
		return "JacketCold"
	case RuleJacketWind:
		return "JacketWind"
	default:
		return fmt.Sprintf("RuleID(%d)", r)
	}
}

func (a Advice) NeedsUmbrella() bool { return a.has(RuleUmbrella) }
func (a Advice) NeedsJacket() bool {
	return a.has(RuleJacketCold) || a.has(RuleJacketWind)
}
func (a Advice) has(id RuleID) bool {
	for _, t := range a.Triggers {
		if t.Rule == id {
			return true
		}
	}
	return false
}

func DefaultRules() Rules {
	return Rules{
		UmbrellaAtPct: 50,
		JacketBelowC:  12,
		JacketWindKmh: 30,
	}
}

func (r Rules) Validate() error {
	if r.UmbrellaAtPct < 0 || r.UmbrellaAtPct > 100 {
		return ErrInvalidRule
	}
	if r.JacketWindKmh < 0 {
		return ErrInvalidRule
	}
	return nil
}

func Advise(hours []Hour, r Rules) (Advice, error) {
	if err := r.Validate(); err != nil {
		return Advice{}, err
	}

	if len(hours) == 0 {
		return Advice{}, ErrNoHours
	}

	first := hours[0]

	rainiest := Peak[int]{Value: first.RainChancePct, At: first.At}
	coldest := Peak[float64]{Value: first.TempC, At: first.At}
	windiest := Peak[float64]{Value: first.WindKmh, At: first.At}

	for _, h := range hours[1:] {
		if h.RainChancePct > rainiest.Value {
			rainiest = Peak[int]{Value: h.RainChancePct, At: h.At}
		}
		if h.TempC < coldest.Value {
			coldest = Peak[float64]{Value: h.TempC, At: h.At}
		}
		if h.WindKmh > windiest.Value {
			windiest = Peak[float64]{Value: h.WindKmh, At: h.At}
		}
	}

	var triggers []Trigger

	if rainiest.Value >= r.UmbrellaAtPct {
		triggers = append(triggers, Trigger{
			Rule:      RuleUmbrella,
			Threshold: float64(r.UmbrellaAtPct),
			Observed:  float64(rainiest.Value),
		})
	}
	if coldest.Value < r.JacketBelowC {
		triggers = append(triggers, Trigger{
			Rule:      RuleJacketCold,
			Threshold: r.JacketBelowC,
			Observed:  coldest.Value,
		})
	}
	if windiest.Value >= r.JacketWindKmh {
		triggers = append(triggers, Trigger{
			Rule:      RuleJacketWind,
			Threshold: r.JacketWindKmh,
			Observed:  windiest.Value,
		})
	}

	return Advice{
		Triggers: triggers,
		Rainiest: rainiest,
		Coldest:  coldest,
		Windiest: windiest,
	}, nil
}
