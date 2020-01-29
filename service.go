package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"

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

type ScanningService struct {
	JobGroup sync.WaitGroup
}

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
