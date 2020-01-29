package scanner

import (
	"os"

	"github.com/aquasecurity/harbor-scanner-trivy/pkg/etc"
	"github.com/aquasecurity/harbor-scanner-trivy/pkg/trivy"
)

var cacheDir string = os.Getenv("HOME") + "/.cache"
var reportsDir string = os.Getenv("HOME") + "/.reports"

func initDir() {
	os.MkdirAll(cacheDir, os.ModePerm)
	os.MkdirAll(reportsDir, os.ModePerm)
}

func RunTrivyScan(imageRef string) (trivy.ScanReport, error) {
	initDir()

	w := trivy.NewWrapper(etc.Trivy{CacheDir: cacheDir,
		ReportsDir: reportsDir,
		Severity:   "UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL",
		VulnType:   "os,library"})
	return w.Run(imageRef, trivy.RegistryAuth{})
}
