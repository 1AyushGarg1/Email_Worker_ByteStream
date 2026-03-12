package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1AyushGarg1/EmailWorker/config"
	"github.com/1AyushGarg1/EmailWorker/logger"
	"github.com/1AyushGarg1/EmailWorker/models"
	"github.com/1AyushGarg1/EmailWorker/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// EmailWorker consumes email jobs from RabbitMQ and sends them using an EmailService
type EmailWorker struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	emailService service.EmailService
	logger       *zap.SugaredLogger
}

// NewEmailWorker initializes a new worker
func NewEmailWorker(amqpURL, queueName string, emailService service.EmailService, logger *zap.SugaredLogger) (*EmailWorker, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Ensure the queue exists before consuming from it
	_, err = ch.QueueDeclare(
		queueName, // name (must match publisher)
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Tell RabbitMQ not to give more than 1 message at a time to this worker.
	// This helps with fair dispatching if you run multiple workers.
	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &EmailWorker{
		conn:         conn,
		ch:           ch,
		emailService: emailService,
		logger:       logger,
	}, nil
}

// Start begins listening to the queue and processing emails forever
func (w *EmailWorker) Start(ctx context.Context, queueName string) error {
	msgs, err := w.ch.Consume(
		queueName, // queue
		"",            // consumer tag
		false,         // auto-ack (we use manual ack to guarantee delivery)
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	w.logger.Info("[*] Email Worker started. Waiting for emails to send...")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Context cancelled, shutting down Email Worker.")
			w.Close()
			return nil
		case d, ok := <-msgs:
			if !ok {
				w.logger.Info("RabbitMQ channel closed, shutting down Email Worker.")
				return nil
			}

			w.processMessage(ctx, d)
		}
	}
}

func (w *EmailWorker) processMessage(ctx context.Context, d amqp.Delivery) {
	var job models.EmailJob
	if err := json.Unmarshal(d.Body, &job); err != nil {
		w.logger.Errorf("Failed to unmarshal email job: %v. Raw message: %s", err, string(d.Body))
		// Nack without requeueing to drop the invalid message
		d.Nack(false, false)
		return
	}

	var sendErr error

	// Create a new context with timeout for the actual sending process
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch job.Type {
	case models.EmailTypeOTP:
		// Decode the inner OTPData struct from the interface map
		var otpData models.OTPData
		b, _ := json.Marshal(job.Data)
		json.Unmarshal(b, &otpData)

		sendErr = w.emailService.SendOTP(sendCtx, otpData.RecipientEmail, otpData.OTP)

	case models.EmailTypeTestPaper:
		// Decode the inner TestPaperData struct
		var testData models.TestPaperData
		b, _ := json.Marshal(job.Data)
		json.Unmarshal(b, &testData)

		sendErr = w.emailService.SendMailToStudent(sendCtx, testData.RecipientEmail,testData.UserName, testData.TestPaperTitle, testData.MarksObtained, testData.FeedbackURL)

	default:
		w.logger.Warnf("Unknown email job type: %s", job.Type)
		d.Nack(false, false)
		return
	}

	if sendErr != nil {
		w.logger.Errorf("Failed to send %s email: %v", job.Type, sendErr)
		// Nack and requeue so another worker (or this one) can try again later
		d.Nack(false, true)
	} else {
		w.logger.Infof("Successfully processed %s email.", job.Type)
		// Success! Acknowledge the message so RabbitMQ removes it from the queue
		d.Ack(false)
	}
}

// Close gracefully closes the connections
func (w *EmailWorker) Close() {
	if w.ch != nil {
		w.ch.Close()
	}
	if w.conn != nil {
		w.conn.Close()
	}
}

func main() {
	// Sync loggers on exit
	defer logger.Log.Sync()
	defer logger.Sugar.Sync()

	cfg := config.Cfg

	var emailService service.EmailService
	if cfg.ENV == "production" {
		emailService = service.NewSMTPEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPSender)
	} else {
		emailService = service.NewMockEmailService()
	}

	worker, err := NewEmailWorker(cfg.RabbitMQURL, cfg.RabbitMQQueueName, emailService, logger.Sugar)
	if err != nil {
		logger.Sugar.Fatalf("Failed to initialize EmailWorker: %v", err)
	}
	defer worker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := worker.Start(ctx, cfg.RabbitMQQueueName); err != nil {
			logger.Sugar.Errorf("Worker start failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the worker
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Sugar.Info("Shutting down worker...")
	cancel()

	// Give it a moment to finish processing
	time.Sleep(2 * time.Second)
	logger.Sugar.Info("Worker stopped gracefully.")
}
