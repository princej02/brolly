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

func Advise(hours []Hour, r Rules) (Advice, error) {
	if len(hours) == 0 {
		return Advice{}, ErrNoHours
	}

	return Advice{}, errors.New("not implemented")
}
