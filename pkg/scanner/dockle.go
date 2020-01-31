package scanner

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type DockleIssue struct {
	Code   string   `json:"code"`
	Title  string   `json:"title"`
	Level  string   `json:"level"`
	Alerts []string `json:"alerts"`
}

type DockleReport struct {
	Summary map[string]int `json:"summary"`
	Details []DockleIssue  `json:"details"`
}

// This is from Trivy wrapper
func RunDockleScan(imageRef string) (report DockleReport, err error) {
	log.Debugf("Running dockle scan on: %s", imageRef)

	executable, err := exec.LookPath("dockle")
	if err != nil {
		log.Debugf("Failed to lookup dockle executable")
		return report, err
	}

	reportFile, err := ioutil.TempFile("/tmp", "dockle_report_*.json")
	if err != nil {
		log.Debugf("Failed to create tmpfile: %#v", err)
		return report, err
	}

	args := []string{
		"--format", "json",
		"--output", reportFile.Name(),
		imageRef,
	}

	cmd := exec.Command(executable, args...)

	stderrBuffer := bytes.Buffer{}
	cmd.Stderr = &stderrBuffer

	stdout, err := cmd.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"image_ref": imageRef,
			"exit_code": cmd.ProcessState.ExitCode(),
			"std_err":   stderrBuffer.String(),
			"std_out":   string(stdout),
		}).Error("Running dockle failed")
		return report, xerrors.Errorf("running dockle: %v: %v", err, stderrBuffer.String())
	}

	log.WithFields(log.Fields{
		"image_ref": imageRef,
		"exit_code": cmd.ProcessState.ExitCode(),
		"std_err":   stderrBuffer.String(),
		"std_out":   string(stdout),
	}).Debug("Running dockle finished")

	report, err = parseScanReports(reportFile)
	return
}

func parseScanReports(reportFile io.Reader) (report DockleReport, err error) {
	err = json.NewDecoder(reportFile).Decode(&report)

	if err != nil {
		log.Debugf("Failed to parse dockle JSON report: %#v", err)
		return report, err
	}

	return
}
