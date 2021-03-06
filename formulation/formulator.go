// Package formulation provides a library for automatically formulating queries.
package formulation

import (
	"fmt"
	"github.com/hscells/cqr"
	"github.com/hscells/groove/pipeline"
	"github.com/hscells/groove/stats"
	"github.com/hscells/trecresults"
	"strconv"
)

// Formulator formulates queries to some specification.
type Formulator interface {
	Formulate() ([]cqr.CommonQueryRepresentation, []SupplementalData, error)
	Method() string
	Topic() string
}

// Data contains the actual data saved and how to write it to disk.
type Data struct {
	Name  string
	Value DataMarshaler
}

// DataMarshaler describes how data should be written to disk.
type DataMarshaler interface {
	Marshal() ([]byte, error)
}

// SupplementalData is extra data than may output from the formulation process.
type SupplementalData struct {
	Name string
	Data []Data
}

// ConceptualFormulator formulates queries using the title or string of a systematic review.
type ConceptualFormulator struct {
	title string
	topic string

	LogicComposer
	EntityExtractor
	EntityExpander
	KeywordMapper
	postProcessing []PostProcess
}

// ObjectiveFormulator formulates queries according to the objective approach.
// This implementation writes files to disk as a side effect which can be later be used for analysis.
type ObjectiveFormulator struct {
	seed                                   int
	Folder, Pubdates, SemTypes, MetaMapURL string
	query                                  pipeline.Query
	s                                      stats.EntrezStatisticsSource
	qrels                                  trecresults.Qrels
	MeSHK                                  []int
	DevK, PopK                             []float64

	population     BackgroundCollection
	splitter       Splitter
	analyser       TermAnalyser
	postProcessing []PostProcess
}

type ObjectiveOption func(o *ObjectiveFormulator)

func ObjectiveGrid(devK, popK []float64, meshK []int) ObjectiveOption {
	return func(o *ObjectiveFormulator) {
		o.DevK = devK
		o.PopK = popK
		o.MeSHK = meshK
	}
}

func ObjectiveSplitter(spitter Splitter) ObjectiveOption {
	return func(o *ObjectiveFormulator) {
		o.splitter = spitter
	}
}

func ObjectiveAnalyser(analyser TermAnalyser) ObjectiveOption {
	return func(o *ObjectiveFormulator) {
		o.analyser = analyser
	}
}

func ObjectivePostProcessing(processes ...PostProcess) ObjectiveOption {
	return func(o *ObjectiveFormulator) {
		o.postProcessing = processes
	}
}

func NewObjectiveFormulator(query pipeline.Query, s stats.EntrezStatisticsSource, qrels trecresults.Qrels, population BackgroundCollection, folder, pubdates, semTypes, metamapURL string, options ...ObjectiveOption) *ObjectiveFormulator {
	o := &ObjectiveFormulator{
		s:          s,
		qrels:      qrels,
		population: population,
		query:      query,
		Folder:     folder,
		Pubdates:   pubdates,
		SemTypes:   semTypes,
		MetaMapURL: metamapURL,
		DevK:       []float64{0.20},
		PopK:       []float64{0.02},
		MeSHK:      []int{20},
		splitter:   RandomSplitter(1000),
		analyser:   TermFrequencyAnalyser,
		//DevK:       []float64{0.05, 0.10, 0.15, 0.20, 0.25, 0.30},
		//PopK:       []float64{0.001, 0.01, 0.02, 0.05, 0.10, 0.20},
		//MeSHK:      []int{1, 5, 10, 15, 20, 25},
	}

	for _, option := range options {
		option(o)
	}

	return o
}

// Formulate returns two queries: one with MeSH terms and one without. It also returns the set of unseen documents for evaluation later.
func (o ObjectiveFormulator) Formulate() ([]cqr.CommonQueryRepresentation, []SupplementalData, error) {
	// Identify the relevant studies using relevance assessments.
	var docs []int
	var nonrel []*trecresults.Qrel
	for _, rel := range o.qrels {
		if rel.Score > 0 {
			v, err := strconv.Atoi(rel.DocId)
			if err != nil {
				panic(err)
			}
			docs = append(docs, v)
		} else {
			nonrel = append(nonrel, rel)
		}
	}

	if len(docs) <= 50 {
		return nil, nil, fmt.Errorf("not enough relevant studies (minimmum 50)")
	}

	// Fetch those relevant documents.
	test, err := fetchDocuments(docs, o.s)
	if err != nil {
		panic(err)
	}

	// Split the 'test' set into dev, val, and unseen.
	dev, val, unseen := o.splitter.Split(test)
	fmt.Println(len(dev), len(val), len(unseen))

	// Perform 'term frequency analysis' on the development set.
	devTerms, err := o.analyser(dev)
	if err != nil {
		panic(err)
	}

	q1, q2, err := o.derive(devTerms, dev, val, o.population)

	// Post-Processing.
	for _, postProcessor := range o.postProcessing {
		q1, err = postProcessor(q1)
		if err != nil {
			return nil, nil, err
		}
		q2, err = postProcessor(q2)
		if err != nil {
			return nil, nil, err
		}
	}

	sup := SupplementalData{
		Name: "objective",
		Data: []Data{
			{
				Name:  "unseen.qrels",
				Value: MakeQrels(unseen),
			},
		},
	}

	return []cqr.CommonQueryRepresentation{q1, q2}, []SupplementalData{sup}, nil
}

func (o ObjectiveFormulator) Method() string {
	return "objective"
}

func (o ObjectiveFormulator) Topic() string {
	return o.query.Topic
}

func NewConceptualFormulator(title, topic string, logicComposer LogicComposer, entityExtractor EntityExtractor, entityExpander EntityExpander, keywordMapper KeywordMapper, postProcessing ...PostProcess) *ConceptualFormulator {
	return &ConceptualFormulator{
		title:           title,
		topic:           topic,
		LogicComposer:   logicComposer,
		EntityExtractor: entityExtractor,
		EntityExpander:  entityExpander,
		KeywordMapper:   keywordMapper,
		postProcessing:  postProcessing,
	}
}

func (t ConceptualFormulator) Formulate() ([]cqr.CommonQueryRepresentation, []SupplementalData, error) {
	// Query Logic Composition.
	q, err := t.LogicComposer.Compose(t.title)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("compose", q)

	// Entity Extraction.
	q, err = t.EntityExtractor.Extract(q)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("extract", q)

	// Entity Expansion.
	if t.EntityExpander != nil {
		q, err = EntityExpansion(q, t.EntityExpander)
		if err != nil {
			return nil, nil, err
		}
	}

	fmt.Println("expand", q)

	// Entities to Keywords Mapping.
	q, err = MapKeywords(q, t.KeywordMapper)
	if err != nil {
		return nil, nil, err
	}

	fmt.Println("mapped", q)

	// Post-Processing.
	for _, postProcessor := range t.postProcessing {
		q, err = postProcessor(q)
		if err != nil {
			return nil, nil, err
		}
	}

	fmt.Println("post-processed", q)

	return []cqr.CommonQueryRepresentation{q}, nil, nil
}

func (t ConceptualFormulator) Method() string {
	return "conceptual"
}

func (t ConceptualFormulator) Topic() string {
	return t.topic
}
