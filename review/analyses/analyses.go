/*
Copyright 2015 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package analyses defines the internal representation of static analysis reports.
package analyses

import (
	"encoding/json"
	"github.com/akatrevorjay/git-appraise/repository"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
)

const (
	// Ref defines the git-notes ref that we expect to contain analysis reports.
	Ref = "refs/notes/devtools/analyses"

	// StatusLooksGoodToMe is the status string representing that analyses reported no messages.
	StatusLooksGoodToMe = "lgtm"
	// StatusForYourInformation is the status string representing that analyses reported informational messages.
	StatusForYourInformation = "fyi"
	// StatusNeedsMoreWork is the status string representing that analyses reported error messages.
	StatusNeedsMoreWork = "nmw"

	// FormatVersion defines the latest version of the request format supported by the tool.
	FormatVersion = 0
)

// Report represents a build/test status report generated by analyses tool.
// Every field is optional.
type Report struct {
	Timestamp string `json:"timestamp,omitempty"`
	URL       string `json:"url,omitempty"`
	Status    string `json:"status,omitempty"`
	// Version represents the version of the metadata format.
	Version int `json:"v,omitempty"`
}

// LocationRange represents the location within a source file that an analysis message covers.
type LocationRange struct {
	StartLine int `json:"start_line,omitempty"`
}

// Location represents the location within a source tree that an analysis message covers.
type Location struct {
	Path  string         `json:"path,omitempty"`
	Range *LocationRange `json:"range,omitempty"`
}

// Note represents a single analysis message.
type Note struct {
	Location    *Location `json:"location,omitempty"`
	Category    string    `json:"category,omitempty"`
	Description string    `json:"description"`
}

// AnalyzeResponse represents the response from a static-analysis tool.
type AnalyzeResponse struct {
	Notes []Note `json:"note,omitempty"`
}

// ReportDetails represents an entire static analysis run (which might include multiple analysis tools).
type ReportDetails struct {
	AnalyzeResponse []AnalyzeResponse `json:"analyze_response,omitempty"`
}

// GetLintReportResult downloads the details of a lint report and returns the responses embedded in it.
func (analysesReport Report) GetLintReportResult() ([]AnalyzeResponse, error) {
	if analysesReport.URL == "" {
		return nil, nil
	}
	res, err := http.Get(analysesReport.URL)
	if err != nil {
		return nil, err
	}
	analysesResults, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	var details ReportDetails
	err = json.Unmarshal([]byte(analysesResults), &details)
	if err != nil {
		return nil, err
	}
	return details.AnalyzeResponse, nil
}

// GetNotes downloads the details of an analyses report and returns the notes embedded in it.
func (analysesReport Report) GetNotes() ([]Note, error) {
	reportResults, err := analysesReport.GetLintReportResult()
	if err != nil {
		return nil, err
	}
	var reportNotes []Note
	for _, reportResult := range reportResults {
		reportNotes = append(reportNotes, reportResult.Notes...)
	}
	return reportNotes, nil
}

// Parse parses an analysis report from a git note.
func Parse(note repository.Note) (Report, error) {
	bytes := []byte(note)
	var report Report
	err := json.Unmarshal(bytes, &report)
	return report, err
}

// GetLatestAnalysesReport takes a collection of analysis reports, and returns the one with the most recent timestamp.
func GetLatestAnalysesReport(reports []Report) (*Report, error) {
	timestampReportMap := make(map[int]*Report)
	var timestamps []int

	for _, report := range reports {
		timestamp, err := strconv.Atoi(report.Timestamp)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, timestamp)
		timestampReportMap[timestamp] = &report
	}
	if len(timestamps) == 0 {
		return nil, nil
	}
	sort.Sort(sort.Reverse(sort.IntSlice(timestamps)))
	return timestampReportMap[timestamps[0]], nil
}

// ParseAllValid takes collection of git notes and tries to parse a analyses report
// from each one. Any notes that are not valid analyses reports get ignored.
func ParseAllValid(notes []repository.Note) []Report {
	var reports []Report
	for _, note := range notes {
		report, err := Parse(note)
		if err == nil && report.Version == FormatVersion {
			reports = append(reports, report)
		}
	}
	return reports
}
