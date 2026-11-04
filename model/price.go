package model

import (
	"fmt"

	"github.com/tamnd/olymmath-agent/content"
)

// Price is the price of a model in US dollars per million tokens.
type Price struct {
	Input  float64
	Output float64
}

// Prices are the prices of spec 07.6 (Anthropic, October 2026).
var Prices = map[string]Price{
	Opus:   {4, 20},
	Sonnet: {2, 10},
	Haiku:  {0.10, 0.50},
}

// Cutoffs are the training cutoffs of the models, as a year and a month. The site shows a contamination note on a problem that is older (spec 07.5). A model with no entry gets no note; add its cutoff from its model card.
var Cutoffs = map[string]string{
	Opus: "2026-06",
}

// The factors of the cached input tokens and of the Batch API.
const (
	// CacheReadFactor is the price of a cache read, as a part of the input price.
	CacheReadFactor = 0.1
	// CacheWriteFactor is the price of a cache write with the five-minute lifetime.
	CacheWriteFactor = 1.25
	// BatchFactor is the price of a call of the Batch API, as a part of the normal price.
	BatchFactor = 0.5
)

// Cost gives the cost in US dollars of the tokens of one or more calls of a model.
func Cost(model string, t content.Tokens, batch bool) (float64, error) {
	p, ok := Prices[model]
	if !ok {
		return 0, fmt.Errorf("the model %s has no price; add it to model.Prices", model)
	}
	c := (float64(t.Input)*p.Input +
		float64(t.CacheRead)*p.Input*CacheReadFactor +
		float64(t.CacheWrite)*p.Input*CacheWriteFactor +
		float64(t.Output)*p.Output) / 1e6
	if batch {
		c *= BatchFactor
	}
	return c, nil
}

// Tokens estimates the number of tokens of a text: about four bytes a token for English and TeX.
func Tokens(text string) int { return (len(text) + 3) / 4 }

// Estimate gives the most that a request can cost: its input and its full output budget.
func Estimate(r Request, batch bool) (float64, error) {
	in := Tokens(r.System)
	for _, m := range r.Messages {
		in += Tokens(m.Text)
	}
	return Cost(r.Model, content.Tokens{Input: in, Output: r.MaxTokens}, batch)
}
