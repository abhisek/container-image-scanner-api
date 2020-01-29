package main

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/abhisek/container-image-scanner-api/pkg/scanner"

	"github.com/aquasecurity/harbor-scanner-trivy/pkg/trivy"
	log "github.com/sirupsen/logrus"
)

type ScanRequest struct {
	ImageRef         string `json:"image"`
	RegistryUsername string `json:"username"`
	RegistryPassword string `json:"password"`
}

type ScanReport struct {
	Version         string           `json:"version"`
	TrivyScanReport trivy.ScanReport `json:"report"`
}

// Hash index is not addressable in Go so we use flat key-value pairing
type ScanningService struct {
	statusMap      map[string]string
	reportMap      map[string]ScanReport
	scannerChannel chan ScanJob
}

type ScanJob struct {
	JobID       string
	ScanRequest ScanRequest
}

const (
	SCAN_STATUS_NEW         = "NEW"
	SCAN_STATUS_IN_PROGRESS = "IN-PROGRESS"
	SCAN_STATUS_COMPLETED   = "COMPLETED"
	SCAN_STATUS_ERROR       = "ERROR"
)

func dockerPullImage(imageRef, user, password string) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Errorf("Failed to create docker client: %#v", err)
		return err
	}

	imagePullOptions := types.ImagePullOptions{}

	if len(user) > 0 && len(password) > 0 {
		authConfig := types.AuthConfig{
			Username: user,
			Password: password,
		}

		authJSON, _ := json.Marshal(authConfig)
		authStr := base64.URLEncoding.EncodeToString(authJSON)

		imagePullOptions.RegistryAuth = authStr
	}

	out, err := cli.ImagePull(ctx, imageRef, imagePullOptions)
	if err != nil {
		log.Errorf("Failed to pull docker image: %#v", err)
		return err
	}

	log.Debugf("Successfully pulled docker image from: %s", imageRef)

	out.Close()
	return nil
}

func (ctx *ScanningService) Init() {
	log.Debugf("Scanning service initialized")

	ctx.scannerChannel = make(chan ScanJob, 5)
	ctx.statusMap = make(map[string]string, 0)
	ctx.reportMap = make(map[string]ScanReport, 0)

	go func() {
		for job := range ctx.scannerChannel {
			ctx.processJob(job)
		}
	}()
}

func (ctx *ScanningService) processJob(job ScanJob) {
	log.Debugf("Processing scan job with id: %s", job.JobID)

	ctx.statusMap[job.JobID] = SCAN_STATUS_IN_PROGRESS
	if report, err := ctx.ScanImage(job.ScanRequest); err != nil {
		ctx.statusMap[job.JobID] = SCAN_STATUS_ERROR
	} else {
		ctx.reportMap[job.JobID] = report
		ctx.statusMap[job.JobID] = SCAN_STATUS_COMPLETED
	}

	log.Debugf("Finished processing job with id: %s status: %s",
		job.JobID, ctx.statusMap[job.JobID])
}

// This method is synchronous
func (ctx *ScanningService) ScanImage(req ScanRequest) (ScanReport, error) {
	log.Debugf("Scanning image from: %s", req.ImageRef)

	if err := dockerPullImage(req.ImageRef, req.RegistryUsername, req.RegistryPassword); err != nil {
		return ScanReport{}, err
	}

	log.Debugf("Starting Trivy scan")
	scanReport, err := scanner.RunTrivyScan(req.ImageRef)

	if err != nil {
		log.Errorf("Failed to execute trivy scan: %#v", err)
		return ScanReport{}, err
	}

	log.Debugf("Completed trivy scan, found %d vulnerabilites on: %s",
		len(scanReport.Vulnerabilities), scanReport.Target)

	return ScanReport{Version: "1", TrivyScanReport: scanReport}, nil
}

func (ctx *ScanningService) AsyncScanImage(req ScanRequest) string {
	scanID := uuid.New().String()
	job := ScanJob{
		JobID:       scanID,
		ScanRequest: req}

	log.Debugf("Async submit scan for image: %s id: %s", req.ImageRef, scanID)

	ctx.statusMap[job.JobID] = SCAN_STATUS_NEW
	ctx.scannerChannel <- job

	return scanID
}

func (ctx *ScanningService) GetScanStatus(scanID string) string {
	status, found := ctx.statusMap[scanID]

	if found {
		return status
	}

	return SCAN_STATUS_ERROR
}
