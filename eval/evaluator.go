// Package eval contains implementations of different evaluation measures for information retrieval.
package eval

import (
	"github.com/hscells/trecresults"
)

// Evaluator is an interface for evaluating a retrieved list of documents.
type Evaluator interface {
	Score(results *trecresults.ResultList, qrels trecresults.Qrels) float64
	Name() string
}

// Evaluate scores documents using supplied evaluation measurements.
func Evaluate(evaluators []Evaluator, results *trecresults.ResultList, qrels trecresults.QrelsFile, topic string) map[string]float64 {
	scores := map[string]float64{}

	// When we evaluate we only ever do so on that topic for performance reasons.
	q := qrels.Qrels[topic]

	// When we retrieve documents, evaluate them.
	if len(*results) > 0 {
		for _, evaluator := range evaluators {
			scores[evaluator.Name()] = evaluator.Score(results, q)
		}
	} else {
		// If no documents were retrieved, we score with an empty list.
		for _, evaluator := range evaluators {
			scores[evaluator.Name()] = evaluator.Score(&trecresults.ResultList{}, q)
		}
	}

	return scores
}
