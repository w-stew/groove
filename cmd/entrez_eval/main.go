package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/alexflint/go-arg"
	"github.com/hscells/groove/eval"
	"github.com/hscells/groove/output"
	"github.com/hscells/groove/retrieval"
	"github.com/hscells/groove/stats"
	"github.com/hscells/trecresults"
	"log"
	"os"
	"path"
	"strings"
)

var (
	name    = "entrez_eval"
	version = "28.Mar.2019"
	author  = "Harry Scells"
)

type args struct {
	RelevanceGrade   int64    `help:"Minimum level of relevance to consider" arg:"-l"`
	Evaluation       []string `help:"Which evaluation measures to use" arg:"-e,separate"`
	ResultHandlers   []string `help:"Which run handlers to use" arg:"-r,separate"`
	RunOutput        string   `help:"Name of processed run file" arg:"-o"`
	EvaluationOutput string   `help:"Name of results file" arg:"-q"`
	RunFile          string   `help:"Path to run file" arg:"required,positional"`
	QrelsFile        string   `help:"Path to qrels file" arg:"required,positional"`
}

func (args) Version() string {
	return version
}

func (args) Description() string {
	return fmt.Sprintf(`%s
@ %s
# %s`, name, author, version)
}

type config struct {
	Entrez struct {
		Email string `toml:"email"`
		Tool  string `toml:"tool"`
		Key   string `toml:"key"`
	} `toml:"entrez"`
}

func main() {
	var args args
	arg.MustParse(&args)

	if len(args.Evaluation) == 0 && len(args.ResultHandlers) == 0 {
		log.Fatalln("nothing to do, quitting")
		os.Exit(1)
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.OpenFile(path.Join(dir, ".entrez_eval"), os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		log.Fatalln(err)
	}

	var c config
	_, err = toml.DecodeReader(f, &c)
	if err != nil {
		log.Fatalln(err)
	}

	e, err := stats.NewEntrezStatisticsSource(
		stats.EntrezTool(c.Entrez.Tool),
		stats.EntrezAPIKey(c.Entrez.Key),
		stats.EntrezEmail(c.Entrez.Email),
		stats.EntrezOptions(stats.SearchOptions{
			Size:    100000,
			RunName: "entrez_eval",
		}))
	if err != nil {
		log.Fatalln(err)
	}

	N, err := e.CollectionSize()
	if err != nil {
		log.Fatalln(err)
	}

	resultsHandlers := make(map[string]retrieval.ResultsHandler)
	evaluationMeasures := make(map[string]eval.Evaluator)

	resultsHandlers["deduplicate"] = retrieval.NewDeduplicator(e)

	evaluationMeasures["precision"] = eval.Precision
	evaluationMeasures["recall"] = eval.Recall
	evaluationMeasures["f1"] = eval.F1Measure
	evaluationMeasures["f0.5"] = eval.F05Measure
	evaluationMeasures["f3"] = eval.F3Measure
	evaluationMeasures["nnr"] = eval.NNR
	evaluationMeasures["wss"] = eval.NewWSSEvaluator(N)
	evaluationMeasures["num_ret"] = eval.NumRet
	evaluationMeasures["num_rel"] = eval.NumRel
	evaluationMeasures["num_rel_ret"] = eval.NumRelRet

	eval.RelevanceGrade = args.RelevanceGrade

	r, err := os.OpenFile(args.RunFile, os.O_RDONLY, 0664)
	if err != nil {
		log.Fatalln(err)
	}
	results, err := trecresults.ResultsFromReader(r)
	if err != nil {
		log.Fatalln(err)
	}

	q, err := os.OpenFile(args.QrelsFile, os.O_RDONLY, 0664)
	if err != nil {
		log.Fatalln(err)
	}
	qrels, err := trecresults.QrelsFromReader(q)
	if err != nil {
		log.Fatalln(err)
	}

	evaluation := make(map[string]map[string]float64)
	size := 0
	for k, v := range results.Results {
		log.Println(k)
		// Process all the results handlers first.
		for _, h := range args.ResultHandlers {
			size += v.Len()
			if handler, ok := resultsHandlers[h]; ok {
				err := handler.Handle(&v)
				if err != nil {
					log.Fatalln(err)
				}
				results.Results[k] = v
			}
		}
		// Then move on to perform the evaluation.
		evaluation[k] = make(map[string]float64)
		for _, ev := range args.Evaluation {
			if m, ok := evaluationMeasures[ev]; ok {
				score := m.Score(&v, qrels.Qrels[k])
				evaluation[k][evaluationMeasures[ev].Name()] = score
			}
		}
	}

	if size > 0 {
		t, err := os.OpenFile(args.RunOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			log.Fatalln(err)
		}

		l := make([]string, size)
		i := 0
		for _, list := range results.Results {
			for _, line := range list {
				l[i] = line.String()
			}
		}

		_, err = t.Write([]byte(strings.Join(l, "\n") + "\n"))
		if err != nil {
			log.Fatalln(err)
		}
	}

	v, err := output.JsonEvaluationFormatter(evaluation)
	if err != nil {
		log.Fatalln(err)
	}
	if len(args.EvaluationOutput) > 0 {
		o, err := os.OpenFile(args.EvaluationOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			log.Fatalln(err)
		}
		_, err = o.WriteString(v)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		_, err = os.Stdout.WriteString(v)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
