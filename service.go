package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

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
	Version          string               `json:"version"`
	TrivyScanReport  trivy.ScanReport     `json:"vulnerabilities"`
	DockleScanReport scanner.DockleReport `json:"audit"`
}

// Hash index is not addressable in Go so we use flat key-value pairing
type ScanningService struct {
	store          Persistence
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

var ConfigScanChannelBufferSize = 100

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
	defer out.Close()

	if err != nil {
		log.Errorf("Failed to pull docker image: %#v", err)
		return err
	}

	log.Debugf("Successfully queued pull request for %s to docker daemon", imageRef)
	log.Debugf("Waiting for image pull to finish")

	// We must wait for pull to complete else docker daemon halts pull
	devNull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0755)
	io.Copy(devNull, out)

	return err
}

func dockerReTagImage(sourceImage string) (destImage string, err error) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Errorf("Failed to create docker client: %#v", err)
		return
	}

	rand.Seed(time.Now().UnixNano())
	destImage = fmt.Sprintf("re-tagged-image-%d:1", rand.Int63())

	if err = cli.ImageTag(ctx, sourceImage, destImage); err != nil {
		log.Errorf("Failed to re-tag docker image: %#v", err)
	}

	return
}

func (ctx *ScanningService) Init() {
	log.Debugf("Initalizing Scanning Service")

	ctx.store.Init()
	ctx.scannerChannel = make(chan ScanJob, ConfigScanChannelBufferSize)

	go func() {
		for job := range ctx.scannerChannel {
			ctx.processJob(job)
		}
	}()

	log.Debugf("Scanning service initialized")
}

func (ctx *ScanningService) processJob(job ScanJob) {
	log.Debugf("Processing scan job with id: %s", job.JobID)

	ctx.store.SetScanStatus(job.JobID, SCAN_STATUS_IN_PROGRESS)
	if report, err := ctx.ScanImage(job.ScanRequest); err != nil {
		ctx.store.SetScanStatus(job.JobID, SCAN_STATUS_ERROR)
	} else {
		ctx.store.SetScanReport(job.JobID, report)
		ctx.store.SetScanStatus(job.JobID, SCAN_STATUS_COMPLETED)
	}

	log.Debugf("Finished processing job with id: %s status: %s",
		job.JobID, ctx.store.GetScanStatus(job.JobID))
}

// This method is synchronous
func (ctx *ScanningService) ScanImage(req ScanRequest) (ScanReport, error) {
	log.Debugf("Scanning image from: %s", req.ImageRef)

	if err := dockerPullImage(req.ImageRef, req.RegistryUsername, req.RegistryPassword); err != nil {
		return ScanReport{}, err
	}

	// This re-tagging is required to prevent some security tools to attempt
	// to pull the image from private registry without credentials and fail
	reTaggedImage, err := dockerReTagImage(req.ImageRef)
	if err != nil {
		log.Errorf("Failed to re-tag image, scan may fail for private repositories")
		reTaggedImage = req.ImageRef
	}

	log.Debugf("Starting Trivy scan")
	trivyScanReport, err := scanner.RunTrivyScan(reTaggedImage)

	if err != nil {
		log.Errorf("Failed to execute trivy scan: %#v", err)
	}

	log.Debugf("Completed trivy scan, found %d vulnerabilites on: %s",
		len(trivyScanReport.Vulnerabilities), trivyScanReport.Target)

	log.Debugf("Starting Dockle scan")
	dockleScanReport, err := scanner.RunDockleScan(reTaggedImage)
	if err != nil {
		log.Errorf("Failed to execute dockle scan: %#v", err)
	}

	log.Debugf("Completed dockle scan, found %d issues", len(dockleScanReport.Details))

	return ScanReport{Version: "1",
		TrivyScanReport: trivyScanReport, DockleScanReport: dockleScanReport}, nil
}

func (ctx *ScanningService) AsyncScanImage(req ScanRequest) string {
	scanID := uuid.New().String()
	job := ScanJob{
		JobID:       scanID,
		ScanRequest: req}

	log.Debugf("Async submit scan for image: %s id: %s", req.ImageRef, scanID)

	ctx.store.SetScanStatus(job.JobID, SCAN_STATUS_NEW)
	ctx.scannerChannel <- job

	return scanID
}

func (ctx *ScanningService) GetScanStatus(scanID string) string {
	return ctx.store.GetScanStatus(scanID)
}

func (ctx *ScanningService) GetScanReport(scanID string) (ScanReport, error) {
	return ctx.store.GetScanReport(scanID)
}
