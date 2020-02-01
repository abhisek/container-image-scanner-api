package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v7"
	log "github.com/sirupsen/logrus"
)

type RedisPersistence struct {
	redis *redis.Client
}

var DEFAULT_EXPIRATION time.Duration = 15 * time.Minute

func (client *RedisPersistence) Init() {
	log.Debugf("Initializing Redis client")

	client.redis = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ENDPOINT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0})

	pong, err := client.redis.Ping().Result()
	if err != nil {
		panic(err)
	}

	log.Debugf("Redis client initialized: %s", pong)
}

func (client *RedisPersistence) SetScanStatus(scanID, status string) error {
	key := fmt.Sprintf("scans:%s:status", scanID)
	_, err := client.redis.Set(key, status, 0).Result()

	if err != nil {
		log.Debugf("Failed to set status map in Redis: %#v", err)
	}

	return err
}

func (client *RedisPersistence) SetScanReport(scanID string, report ScanReport) error {
	key := fmt.Sprintf("scans:%s:report", scanID)
	data, err := json.Marshal(report)

	if err != nil {
		log.Debugf("Failed to serialize report to JSON: %#v", err)
		return err
	}

	_, err = client.redis.Set(key, data, DEFAULT_EXPIRATION).Result()

	if err != nil {
		log.Debugf("Failed to set report map in Redis: %#v", err)
	}

	return err
}

func (client *RedisPersistence) GetScanStatus(scanID string) string {
	return SCAN_STATUS_ERROR
}

func (client *RedisPersistence) GetScanReport(scanID string) (report ScanReport, err error) {
	return report, err
}
