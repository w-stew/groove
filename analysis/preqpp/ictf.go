package preqpp

import (
	"github.com/hscells/groove/analysis"
	"github.com/hscells/groove/pipeline"
	"github.com/hscells/groove/stats"
	"math"
)

type avgICTF struct{}

// AvgICTF is similar to idf, however it attempts to take into account the term frequencies. Inverse collection term
// frequency is defined as the ratio of unique terms in the collection to the term frequency of a term in a document,
// logarithmically smoothed.
var AvgICTF = avgICTF{}

func (avgi avgICTF) Name() string {
	return "AvgICTF"
}

func (avgi avgICTF) Execute(q pipeline.Query, s stats.StatisticsSource) (float64, error) {
	terms := analysis.QueryTerms(q.Query)

	if len(terms) == 0 {
		return 0.0, nil
	}

	sumICTF := 0.0
	fields := analysis.QueryFields(q.Query)

	for _, field := range fields {
		W, err := s.VocabularySize(field)
		if err != nil {
			return 0.0, err
		}
		for _, term := range terms {
			tf, err := s.TotalTermFrequency(term, field)
			if err != nil {
				return 0.0, err
			}
			sumICTF += math.Log2(W) - math.Log2(1+tf)
		}
	}
	return (1.0 / float64(len(terms))) * sumICTF, nil
}
