package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"github.com/phuthien0308/ordering-base/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// The name of the S3 bucket to store configurations in.
// Can be overridden by the environment variable CONFIG_BUCKET.
var defaultBucketName = "configs"

func getBucketName() string {
	if b := os.Getenv("CONFIG_BUCKET"); b != "" {
		return b
	}
	return defaultBucketName
}

func main() {
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	logger := simplelog.NewSimpleZapLogger(zapLogger)

	// Initialize Tracing
	shutdown, err := tracing.DefaultGlobalTracer("configsvc", "http://localhost:9411/api/v2/spans")
	if err != nil {
		logger.Fatal(context.TODO(), "failed to init tracer", tags.Error(err))
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			logger.Error(context.Background(), "failed to shutdown tracer", tags.Error(err))
		}
	}()

	// 1. Initialize AWS Configuration
	// By default, this loads ~/.aws/credentials, environment variables
	// like AWS_REGION, and optionally AWS_ENDPOINT_URL (which is great for LocalStack).
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logger.Fatal(context.TODO(), "unable to load AWS SDK config", tags.Error(err))
	}
	otelaws.AppendMiddlewares(&cfg.APIOptions)

	// 2. Initialize S3 Client (Enable Path-Style for LocalStack compatibility)
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	tracer := otel.Tracer("configsvc")
	mux := http.NewServeMux()

	// -------------------------------------------------------------
	// GET /api/v1/configs -> List Services (Prefixes in S3)
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs", withCORS(func(w http.ResponseWriter, r *http.Request) {
		// Use a Delimiter to find "folder" names (which act as our services)
		output, err := s3Client.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{
			Bucket:    aws.String(getBucketName()),
			Delimiter: aws.String("/"),
		})
		if err != nil {
			logger.Error(r.Context(), "S3 List Error", tags.Error(err))
			http.Error(w, "Failed to list services from S3", http.StatusInternalServerError)
			return
		}
		var services []string
		for _, prefix := range output.CommonPrefixes {
			// Strip the trailing slash (e.g., "product-service/" -> "product-service")
			services = append(services, strings.TrimSuffix(*prefix.Prefix, "/"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(services)
	}))

	// -------------------------------------------------------------
	// GET /api/v1/configs/{service}/versions -> List Versions
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs/{service}/versions", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		prefix := fmt.Sprintf("%s/", service)

		// List all files within the service's "folder"
		output, err := s3Client.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{
			Bucket: aws.String(getBucketName()),
			Prefix: aws.String(prefix),
		})
		if err != nil {
			logger.Error(r.Context(), "S3 List Error", tags.Error(err))
			http.Error(w, "Failed to list versions from S3", http.StatusInternalServerError)
			return
		}

		var versions []string
		for _, object := range output.Contents {
			// Extract version tag out of "product-service/v1.0.0.json"
			fileName := strings.TrimPrefix(*object.Key, prefix)
			versionName := strings.TrimSuffix(fileName, ".json")
			if versionName != "" {
				versions = append(versions, versionName)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
	}))

	// -------------------------------------------------------------
	// GET /api/v1/configs/{service}/versions/{version} -> Get Config
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs/{service}/versions/{version}", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		version := r.PathValue("version")

		key := fmt.Sprintf("%s/%s.json", service, version)

		output, err := s3Client.GetObject(r.Context(), &s3.GetObjectInput{
			Bucket: aws.String(getBucketName()),
			Key:    aws.String(key),
		})
		if err != nil {
			logger.Error(r.Context(), "S3 Get Error", tags.Error(err))
			http.Error(w, "Failed to get config from S3", http.StatusNotFound)
			return
		}
		defer output.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, output.Body)
	}))

	// -------------------------------------------------------------
	// PUT /api/v1/configs/{service}/versions/{version} -> Save Config
	// -------------------------------------------------------------
	mux.HandleFunc("PUT /api/v1/configs/{service}/versions/{version}", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		version := r.PathValue("version")

		body, err := io.ReadAll(r.Body)
		if err != nil || !json.Valid(body) {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		key := fmt.Sprintf("%s/%s.json", service, version)

		// Upload payload to S3
		_, err = s3Client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      aws.String(getBucketName()),
			Key:         aws.String(key),
			Body:        bytes.NewReader(body),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			logger.Error(r.Context(), "S3 Put Error", tags.Error(err))
			http.Error(w, "Failed to upload to S3", http.StatusInternalServerError)
			return
		}

		logger.Info(r.Context(), "Successfully wrote configuration", tags.String("bucket", getBucketName()), tags.String("key", key))

		// Return success response to the UI
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))

	// Fallback OPTIONS handler for Preflight CORS
	mux.HandleFunc("/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	logger.Info(context.TODO(), "S3 Config API running on :8089")
	if err := http.ListenAndServe(":8089", otelhttp.NewHandler(mux, "configsvc")); err != nil {
		logger.Fatal(context.TODO(), "server failed", tags.Error(err))
	}
}

// Global CORS Middleware
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
}
