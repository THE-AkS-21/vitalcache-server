package services

import (
	"log/slog"
	"time"
)

// EmailJob defines the data needed to send an email.
type EmailJob struct {
	To             string
	Subject        string
	Body           string
	AttachmentPath string
	Retries        int
}

// EmailJobQueue is a channel that acts as our in-memory queue.
var EmailJobQueue chan EmailJob

// InitEmailQueue creates the channel.
func InitEmailQueue() {
	// A buffered channel that can hold up to 100 jobs
	EmailJobQueue = make(chan EmailJob, 100)
}

// StartEmailWorker launches a goroutine to process jobs from the queue.
func StartEmailWorker() {
	emailService := NewEmailService()

	go func() {
		for job := range EmailJobQueue {
			slog.Info("Worker picked up a new email job", "to", job.To)
			err := emailService.SendEmailWithAttachment(job.To, job.Subject, job.Body, job.AttachmentPath)

			// Implement retry logic with exponential backoff
			if err != nil {
				if job.Retries < 3 { // Attempt a maximum of 3 retries
					job.Retries++
					delay := time.Duration(job.Retries*job.Retries) * time.Second // 1s, 4s, 9s
					slog.Warn("Email failed to send. Re-queuing job.", "to", job.To, "retry_attempt", job.Retries, "delay", delay)
					time.AfterFunc(delay, func() {
						EmailJobQueue <- job
					})
				} else {
					slog.Error("Email job failed after multiple retries", "to", job.To, "error", err)
				}
			}
		}
	}()

	slog.Info("📧 Email worker started successfully")
}
