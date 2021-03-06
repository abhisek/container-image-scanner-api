package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	log "github.com/sirupsen/logrus"
)

type Persistence struct {
	redis *redis.Client
}

var REPORT_EXPIRATION_WINDOW time.Duration = 15 * time.Minute

func (client *Persistence) Init() {
	log.Debugf("Initializing Redis client")

	client.redis = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ENDPOINT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0})

	pong, err := client.redis.Ping().Result()
	if err != nil {
		panic(err)
	}

	if val := os.Getenv("REPORT_EXPIRATION_WINDOW"); len(val) > 0 {
		if valN, err := strconv.Atoi(val); err != nil {
			log.Debugf("Setting REPORT_EXPIRATION_WINDOW: %d", valN)
			REPORT_EXPIRATION_WINDOW = time.Duration(valN) * time.Second
		}
	}

	log.Debugf("Redis client initialized: %s", pong)
}

func (client *Persistence) SetScanStatus(scanID, status string) error {
	key := fmt.Sprintf("scans:%s:status", scanID)
	_, err := client.redis.Set(key, status, 0).Result()

	if err != nil {
		log.Debugf("Failed to set status map in Redis: %#v", err)
	}

	return err
}

func (client *Persistence) SetScanReport(scanID string, report ScanReport) error {
	key := fmt.Sprintf("scans:%s:report", scanID)
	data, err := json.Marshal(report)

	if err != nil {
		log.Debugf("Failed to serialize report to JSON: %#v", err)
		return err
	}

	_, err = client.redis.Set(key, data, REPORT_EXPIRATION_WINDOW).Result()

	if err != nil {
		log.Debugf("Failed to set report map in Redis: %#v", err)
	}

	return err
}

func (client *Persistence) GetScanStatus(scanID string) string {
	key := fmt.Sprintf("scans:%s:status", scanID)
	value, err := client.redis.Get(key).Result()

	if err != nil {
		log.Debugf("Failed to get status from Redis: %#v", err)
		return SCAN_STATUS_ERROR
	}

	return value
}

func (client *Persistence) GetScanReport(scanID string) (report ScanReport, err error) {
	key := fmt.Sprintf("scans:%s:report", scanID)

	value, err := client.redis.Get(key).Result()

	if err != nil {
		log.Debugf("Failed to get report from Redis: %#v", err)
		return report, err
	}

	err = json.Unmarshal([]byte(value), &report)
	if err != nil {
		log.Debugf("Failed to unmarshal report JSON from redis: %#v", err)
	}

	return report, err
}
