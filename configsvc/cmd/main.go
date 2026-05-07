package main

import (
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
	"github.com/google/go-jsonnet"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"github.com/phuthien0308/ordering-base/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	logger := simplelog.NewSimpleLogger(zapLogger)

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
	// GET /api/v1/configs/{service}/versions -> List Template Versions
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs/{service}/versions", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		prefix := fmt.Sprintf("%s/templates/", service)

		// List all files within the service's templates folder
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
			fileName := strings.TrimPrefix(*object.Key, prefix)
			versionName := strings.TrimSuffix(fileName, ".jsonnet")
			if versionName != "" && !strings.HasSuffix(versionName, "/") {
				versions = append(versions, versionName)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
	}))

	// Helper to fetch string from S3
	fetchFromS3 := func(ctx context.Context, key string) (string, error) {
		out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(getBucketName()),
			Key:    aws.String(key),
		})
		if err != nil {
			return "", err
		}
		defer out.Body.Close()
		b, err := io.ReadAll(out.Body)
		return string(b), err
	}

	// -------------------------------------------------------------
	// GET /api/v1/configs/{service}/bundle/{version} -> Get Full State
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs/{service}/bundle/{version}", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		version := r.PathValue("version")

		tmpl, _ := fetchFromS3(r.Context(), fmt.Sprintf("%s/templates/%s.jsonnet", service, version))
		
		// List all value files
		valPrefix := fmt.Sprintf("%s/values/", service)
		valOutput, _ := s3Client.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{
			Bucket: aws.String(getBucketName()),
			Prefix: aws.String(valPrefix),
		})

		valuesMap := make(map[string]string)
		if valOutput != nil {
			for _, object := range valOutput.Contents {
				envName := strings.TrimSuffix(strings.TrimPrefix(*object.Key, valPrefix), ".json")
				if envName != "" && !strings.HasSuffix(envName, "/") {
					valContent, _ := fetchFromS3(r.Context(), *object.Key)
					valuesMap[envName] = valContent
				}
			}
		}

		// Ensure defaults
		if tmpl == "" {
			tmpl = "{\n  \n}"
		}
		if _, ok := valuesMap["base"]; !ok {
			valuesMap["base"] = "{}"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"template": tmpl,
			"values":   valuesMap,
		})
	}))

	// -------------------------------------------------------------
	// PUT /api/v1/configs/{service}/bundle/{version} -> Save Full State
	// -------------------------------------------------------------
	mux.HandleFunc("PUT /api/v1/configs/{service}/bundle/{version}", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		version := r.PathValue("version")

		type BundleRequest struct {
			Template string            `json:"template"`
			Values   map[string]string `json:"values"` // base, dev, prod, etc.
		}

		var bundle BundleRequest
		if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Helper to write to S3
		writeToS3 := func(key, content, contentType string) error {
			_, err := s3Client.PutObject(r.Context(), &s3.PutObjectInput{
				Bucket:      aws.String(getBucketName()),
				Key:         aws.String(key),
				Body:        strings.NewReader(content),
				ContentType: aws.String(contentType),
			})
			if err == nil {
				logger.Info(r.Context(), "Successfully wrote", tags.String("key", key))
			}
			return err
		}

		// Write template
		tmplKey := fmt.Sprintf("%s/templates/%s.jsonnet", service, version)
		if err := writeToS3(tmplKey, bundle.Template, "text/plain"); err != nil {
			http.Error(w, "Failed to save template", http.StatusInternalServerError)
			return
		}

		// Write values
		for env, val := range bundle.Values {
			valKey := fmt.Sprintf("%s/values/%s.json", service, env)
			if err := writeToS3(valKey, val, "application/json"); err != nil {
				http.Error(w, "Failed to save values for "+env, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))

	// -------------------------------------------------------------
	// POST /api/v1/configs/render -> Render Jsonnet live
	// -------------------------------------------------------------
	mux.HandleFunc("POST /api/v1/configs/render", withCORS(func(w http.ResponseWriter, r *http.Request) {
		type RenderRequest struct {
			Template   string `json:"template"`
			BaseValues string `json:"base_values"`
			EnvValues  string `json:"env_values"`
		}

		var reqPayload RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Pre-process: Merge BaseValues and EnvValues using Jsonnet!
		base := reqPayload.BaseValues
		if base == "" {
			base = "{}"
		}
		env := reqPayload.EnvValues
		if env == "" {
			env = "{}"
		}

		mergeVM := jsonnet.MakeVM()
		mergeVM.ExtCode("base", base)
		mergeVM.ExtCode("env", env)
		mergedJSON, err := mergeVM.EvaluateAnonymousSnippet("merge.jsonnet", "std.extVar('base') + std.extVar('env')")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to merge values: " + err.Error()})
			return
		}

		// Create the main Jsonnet VM for the template
		vm := jsonnet.MakeVM()
		vm.ExtCode("values", mergedJSON)

		// Evaluate the template
		jsonOutput, err := vm.EvaluateAnonymousSnippet("template.jsonnet", reqPayload.Template)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Template error: " + err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonOutput))
	}))

	// -------------------------------------------------------------
	// GET /api/v1/configs/{service}/render/{version}/{env} -> For Microservices
	// -------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/configs/{service}/render/{version}/{env}", withCORS(func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		version := r.PathValue("version")
		envName := r.PathValue("env")

		// 1. Fetch Template
		tmplKey := fmt.Sprintf("%s/templates/%s.jsonnet", service, version)
		tmpl, err := fetchFromS3(r.Context(), tmplKey)
		if err != nil {
			http.Error(w, "Template not found", http.StatusNotFound)
			return
		}

		// 2. Fetch Base Values
		baseKey := fmt.Sprintf("%s/values/base.json", service)
		base, _ := fetchFromS3(r.Context(), baseKey)
		if base == "" {
			base = "{}"
		}

		// 3. Fetch Env Values
		envKey := fmt.Sprintf("%s/values/%s.json", service, envName)
		env, _ := fetchFromS3(r.Context(), envKey)
		if env == "" {
			env = "{}"
		}

		// Merge values
		mergeVM := jsonnet.MakeVM()
		mergeVM.ExtCode("base", base)
		mergeVM.ExtCode("env", env)
		mergedJSON, err := mergeVM.EvaluateAnonymousSnippet("merge", "std.extVar('base') + std.extVar('env')")
		if err != nil {
			http.Error(w, "Failed to merge base and env values", http.StatusInternalServerError)
			return
		}

		// Render Template
		vm := jsonnet.MakeVM()
		vm.ExtCode("values", mergedJSON)
		jsonOutput, err := vm.EvaluateAnonymousSnippet("template", tmpl)
		if err != nil {
			http.Error(w, "Template evaluation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonOutput))
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
