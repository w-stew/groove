package query

import (
	"encoding/xml"
	"github.com/hscells/groove/learning/seed"
	"github.com/hscells/groove/pipeline"
	"github.com/hscells/groove/stats"
	"io/ioutil"
)

// ProtocolQuerySource loads systematic review protocols from XML files that
// follow the structure of:
//
// ```
//<?xml version='1.0' encoding='UTF-8'?>
//<root>
//<objective>
//Our primary objective was to assess the diagnostic accuracy of galactomannan detection in serum for the diagnosis of invasive aspergillosis in immunocompromised patients, at different cut-off values for test positivity.
//Secondary objectives
//We aimed to study several possible sources of heterogeneity: subgroups of patients, different interpretations of the EORTC/MSG criteria as the reference standard and study design features.</objective>
//<type_of_study>
//Studies that assessed the diagnostic accuracy of galactomannan detection by the Platelia© sandwich ELISA test, with either prospective or retrospective data collection, were eligible. The galactomannan ELISA could be assessed alone or in comparison to other tests.</type_of_study>
//<participants>
//Studies had to include patients with neutropenia or patients whose neutrophils are functionally compromised. We included studies with the following patient groups:
//patients with haematological malignancies, receiving haematopoietic stem cell transplants, chemotherapeutics or immunosuppressive drugs;
//solid organ transplant recipients and other patients who are receiving immunosuppressive drugs for a prolonged time;
//patients with cancer who are receiving chemotherapeutics;
//patients with a medical condition compromising the immune system, such as HIV/AIDS and chronic granulomatous disease (CGD, an inherited abnormality of the neutrophils).</participants>
//<index_tests>
//A commercially available galactomannan sandwich ELISA (Platelia©) was the test under evaluation. We only included studies concerning galactomannan detection in serum. We excluded studies addressing detection in BAL fluid, a number of other body fluids, such as CSF or peritoneal fluid, and tissue. We also excluded studies evaluating in-house serum galactomannan tests.</index_tests>
//<target_conditions>
//The target condition of this review was invasive aspergillosis, also called invasive pulmonary aspergillosis or systemic aspergillosis.</target_conditions>
//<reference_standards>
//The following reference standards can be used to define the target condition:
//autopsy;
//the criteria of the EORTC/MSG (Ascioglu 2002; De Pauw 2008); or
//the demonstration of hyphal invasion in biopsies, combined with a positive culture for Aspergillus species from the same specimens.
//The gold standard for this diagnosis is autopsy, combined with a positive culture of Aspergillus species from the autopsy specimens, or with histopathological evidence of Aspergillus. Autopsy is rarely reported, therefore we decided to take the criteria of the EORTC/MSG as the reference standard. These criteria divide the patient population into four categories: patients with proven invasive aspergillosis, patients who probably have invasive aspergillosis, patients who possibly have invasive aspergillosis and patients without invasive aspergillosis (see Table 1). This division is based on host factor criteria, microbiological criteria and clinical criteria. Clinical studies have shown that these criteria do not match autopsy results perfectly. This especially true for the possible category. For clinical trials investigating the effect of treatment, for example, it is recommended that only the proven and probable categories are used (Borlenghi 2007; Subira 2003).
//The exclusion of patients with 'possible' invasive aspergillosis, which can be regarded as group of 'difficult or atypical' patients, is likely to affect the observed diagnostic accuracy of a test. Also, the exclusion of any other of the reference standard groups may affect the accuracy of the index test. We therefore excluded studies explicitly excluding one of the four categories of patients from the review, as well as studies in which it is not clear how many patients with proven, probable, possible or no invasive aspergillosis had positive or negative index test results.</reference_standards>
//</root>
// ```
//
// The source then generates queries according to the package github.com/hscells/groove/learning/seed.
type ProtocolQuerySource struct {
}

// QuickUMLSProtocolQuerySource uses QuickUMLS to perform additional steps in the query formulation process.
type QuickUMLSProtocolQuerySource struct {
	threshold float64
	url       string
	ss        stats.StatisticsSource
}

// protocol is a representation of a systematic review protocol in XML.
type protocol struct {
	Objective          string `xml:"objective"`
	TypeOfStudy        string `xml:"type_of_study"`
	Participants       string `xml:"participants"`
	IndexTests         string `xml:"index_tests"`
	TargetConditions   string `xml:"target_conditions"`
	ReferenceStandards string `xml:"reference_standards"`
}

func (ProtocolQuerySource) Load(directory string) ([]pipeline.Query, error) {
	// First, get a list of files in the directory.
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	// Next, read all files, generating queries for each file.
	var queries []pipeline.Query
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if len(f.Name()) == 0 {
			continue
		}

		source, err := ioutil.ReadFile(directory + "/" + f.Name())
		if err != nil {
			return nil, err
		}

		var p protocol
		err = xml.Unmarshal(source, &p)
		if err != nil {
			return nil, err
		}

		c := seed.NewProtocolConstructor(p.Objective, p.Participants, p.IndexTests, p.TargetConditions)
		q, err := c.Construct()
		if err != nil {
			return nil, err
		}
		for _, query := range q {
			queries = append(queries, pipeline.NewQuery(f.Name(), f.Name(), query))
		}
	}
	return queries, nil
}

func NewProtocolQuerySource() ProtocolQuerySource {
	return ProtocolQuerySource{}
}

func (q QuickUMLSProtocolQuerySource) Load(directory string) ([]pipeline.Query, error) {
	// First, get a list of files in the directory.
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	// Next, read all files, generating queries for each file.
	var queries []pipeline.Query
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if len(f.Name()) == 0 {
			continue
		}

		source, err := ioutil.ReadFile(directory + "/" + f.Name())
		if err != nil {
			return nil, err
		}

		var p protocol
		err = xml.Unmarshal(source, &p)
		if err != nil {
			return nil, err
		}

		c := seed.NewQuickUMLSProtocolConstructor(p.Objective, p.Participants, p.IndexTests, p.TargetConditions, q.url, q.threshold, q.ss)
		q, err := c.Construct()
		if err != nil {
			return nil, err
		}
		for _, query := range q {
			queries = append(queries, pipeline.NewQuery(f.Name(), f.Name(), query))
		}
	}
	return queries, nil
}

func NewQuickUMLSProtocolQuerySource(url string, ss stats.StatisticsSource, threshold float64) QuickUMLSProtocolQuerySource {
	return QuickUMLSProtocolQuerySource{
		url:       url,
		ss:        ss,
		threshold: threshold,
	}
}
