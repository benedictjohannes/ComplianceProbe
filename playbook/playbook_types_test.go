package playbook

import (
	"testing"
)

func TestAssertion_GetMinPassingScore(t *testing.T) {
	one := 1
	five := 5
	zero := 0
	tests := []struct {
		name string
		a    Assertion
		want int
	}{
		{
			name: "nil score with no cmds returns 1 (fallback)",
			a:    Assertion{MinPassingScore: nil, Cmds: nil},
			want: 1,
		},
		{
			name: "nil score with one default cmd returns 1",
			a:    Assertion{MinPassingScore: nil, Cmds: []Cmd{{}}},
			want: 1,
		},
		{
			name: "nil score with multiple default cmds returns their count",
			a:    Assertion{MinPassingScore: nil, Cmds: []Cmd{{}, {}, {}}},
			want: 3,
		},
		{
			name: "nil score with weighted cmds returns their sum",
			a:    Assertion{MinPassingScore: nil, Cmds: []Cmd{{PassScore: &five}, {PassScore: &one}}},
			want: 6,
		},
		{
			name: "nil score with zero-score cmds returns 1 (fallback)",
			a:    Assertion{MinPassingScore: nil, Cmds: []Cmd{{PassScore: &zero}, {PassScore: &zero}}},
			want: 1,
		},
		{
			name: "explicit 1 returns 1 even if sum is different",
			a:    Assertion{MinPassingScore: &one, Cmds: []Cmd{{}, {}}},
			want: 1,
		},
		{
			name: "explicit 5 returns 5",
			a:    Assertion{MinPassingScore: &five},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.GetMinPassingScore(); got != tt.want {
				t.Errorf("Assertion.GetMinPassingScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCmd_GetPassScore(t *testing.T) {
	one := 1
	zero := 0
	tests := []struct {
		name string
		c    Cmd
		want int
	}{
		{
			name: "nil score returns default 1",
			c:    Cmd{PassScore: nil},
			want: 1,
		},
		{
			name: "explicit 1 returns 1",
			c:    Cmd{PassScore: &one},
			want: 1,
		},
		{
			name: "explicit 0 returns 0",
			c:    Cmd{PassScore: &zero},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.GetPassScore(); got != tt.want {
				t.Errorf("Cmd.GetPassScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCmd_GetFailScore(t *testing.T) {
	minusOne := -1
	zero := 0
	tests := []struct {
		name string
		c    Cmd
		want int
	}{
		{
			name: "nil score returns default -1",
			c:    Cmd{FailScore: nil},
			want: -1,
		},
		{
			name: "explicit -1 returns -1",
			c:    Cmd{FailScore: &minusOne},
			want: -1,
		},
		{
			name: "explicit 0 returns 0",
			c:    Cmd{FailScore: &zero},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.GetFailScore(); got != tt.want {
				t.Errorf("Cmd.GetFailScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluationRule_GetIncludeStdErr(t *testing.T) {
	trueVal := true
	falseVal := false
	tests := []struct {
		name string
		r    EvaluationRule
		want bool
	}{
		{
			name: "nil returns default false",
			r:    EvaluationRule{IncludeStdErr: nil},
			want: false,
		},
		{
			name: "explicit true returns true",
			r:    EvaluationRule{IncludeStdErr: &trueVal},
			want: true,
		},
		{
			name: "explicit false returns false",
			r:    EvaluationRule{IncludeStdErr: &falseVal},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetIncludeStdErr(); got != tt.want {
				t.Errorf("EvaluationRule.GetIncludeStdErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGatherSpec_GetIncludeStdErr(t *testing.T) {
	trueVal := true
	falseVal := false
	tests := []struct {
		name string
		g    GatherSpec
		want bool
	}{
		{
			name: "nil returns default false",
			g:    GatherSpec{IncludeStdErr: nil},
			want: false,
		},
		{
			name: "explicit true returns true",
			g:    GatherSpec{IncludeStdErr: &trueVal},
			want: true,
		},
		{
			name: "explicit false returns false",
			g:    GatherSpec{IncludeStdErr: &falseVal},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.GetIncludeStdErr(); got != tt.want {
				t.Errorf("GatherSpec.GetIncludeStdErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
