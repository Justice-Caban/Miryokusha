package suwayomi

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// RuntimeValidator validates the schema at runtime and caches results
type RuntimeValidator struct {
	client          *Client
	expectedSchema  *IntrospectionResult
	lastValidation  *SchemaValidationResult
	lastValidatedAt time.Time
	mu              sync.RWMutex
	validateOnce    sync.Once
}

// NewRuntimeValidator creates a new runtime validator
func NewRuntimeValidator(client *Client) *RuntimeValidator {
	return &RuntimeValidator{
		client: client,
	}
}

// ValidateOnStartup validates the schema on application startup
// This should be called once during app initialization
func (rv *RuntimeValidator) ValidateOnStartup(schemaPath string) error {
	var validationErr error

	rv.validateOnce.Do(func() {
		log.Println("🔍 Validating GraphQL schema...")

		// Try to load expected schema
		expectedSchema, err := LoadSchemaFromFile(schemaPath)
		if err != nil {
			// If no schema file exists, introspect and save it
			log.Printf("⚠️  No expected schema found at %s, introspecting from server...", schemaPath)

			schema, err := rv.client.IntrospectSchema()
			if err != nil {
				validationErr = fmt.Errorf("failed to introspect schema: %w", err)
				return
			}

			// Save as baseline for future validation
			if err := schema.SaveSchemaToFile(schemaPath); err != nil {
				log.Printf("⚠️  Could not save baseline schema: %v", err)
			} else {
				log.Printf("✅ Baseline schema saved to %s", schemaPath)
			}

			rv.expectedSchema = schema
			rv.lastValidatedAt = time.Now()
			log.Println("✅ Schema validation skipped (baseline created)")
			return
		}

		rv.expectedSchema = expectedSchema

		// Introspect actual server schema
		actualSchema, err := rv.client.IntrospectSchema()
		if err != nil {
			validationErr = fmt.Errorf("failed to introspect server schema: %w", err)
			return
		}

		// Validate
		validator := NewSchemaValidator(expectedSchema, actualSchema)
		result := validator.Validate()

		rv.mu.Lock()
		rv.lastValidation = result
		rv.lastValidatedAt = time.Now()
		rv.mu.Unlock()

		// Log results
		if result.IsValid {
			log.Println("✅ Schema validation passed")
			if len(result.Warnings) > 0 {
				log.Printf("⚠️  %d warnings found:", len(result.Warnings))
				for _, warning := range result.Warnings {
					log.Printf("   %s", warning)
				}
			}
		} else {
			log.Println("❌ Schema validation failed")
			log.Printf("Errors: %d, Warnings: %d", len(result.Errors), len(result.Warnings))

			// Log first few errors
			maxErrors := 5
			for i, err := range result.Errors {
				if i >= maxErrors {
					log.Printf("   ... and %d more errors", len(result.Errors)-maxErrors)
					break
				}
				log.Printf("   ❌ %s", err)
			}

			// Set error
			validationErr = fmt.Errorf("schema validation failed with %d errors", len(result.Errors))
		}
	})

	return validationErr
}

// ValidateInBackground performs validation in a background goroutine
func (rv *RuntimeValidator) ValidateInBackground(schemaPath string) {
	go func() {
		if err := rv.ValidateOnStartup(schemaPath); err != nil {
			log.Printf("⚠️  Background schema validation failed: %v", err)
			log.Println("⚠️  Application will continue, but API calls may fail")
		}
	}()
}

// GetLastValidation returns the last validation result (thread-safe)
func (rv *RuntimeValidator) GetLastValidation() (*SchemaValidationResult, time.Time) {
	rv.mu.RLock()
	defer rv.mu.RUnlock()
	return rv.lastValidation, rv.lastValidatedAt
}

// IsValid returns whether the last validation passed
func (rv *RuntimeValidator) IsValid() bool {
	rv.mu.RLock()
	defer rv.mu.RUnlock()
	return rv.lastValidation != nil && rv.lastValidation.IsValid
}

// GetValidationReport returns a human-readable validation report
func (rv *RuntimeValidator) GetValidationReport() string {
	rv.mu.RLock()
	defer rv.mu.RUnlock()

	if rv.lastValidation == nil {
		return "Schema validation has not been performed yet"
	}

	return rv.lastValidation.Report()
}

// SchemaCompatibilityMode determines how strict schema validation should be
type SchemaCompatibilityMode int

const (
	// CompatibilityStrict fails if schema doesn't match exactly
	CompatibilityStrict SchemaCompatibilityMode = iota
	// CompatibilityWarn logs warnings but continues
	CompatibilityWarn
	// CompatibilityIgnore skips validation entirely
	CompatibilityIgnore
)

// ValidateWithMode validates schema with specified compatibility mode
func ValidateWithMode(client *Client, schemaPath string, mode SchemaCompatibilityMode) error {
	if mode == CompatibilityIgnore {
		return nil
	}

	validator := NewRuntimeValidator(client)
	err := validator.ValidateOnStartup(schemaPath)

	if err != nil {
		if mode == CompatibilityWarn {
			log.Printf("⚠️  Schema validation failed but continuing (warn mode): %v", err)
			return nil
		}
		return err
	}

	return nil
}

// GetSchemaCompatibilityMode reads the schema compatibility mode from environment
func GetSchemaCompatibilityMode() SchemaCompatibilityMode {
	mode := os.Getenv("SUWAYOMI_SCHEMA_VALIDATION")
	switch mode {
	case "strict":
		return CompatibilityStrict
	case "warn":
		return CompatibilityWarn
	case "ignore":
		return CompatibilityIgnore
	default:
		// Default to warn mode for development
		return CompatibilityWarn
	}
}

// AutoValidateSchema is a helper that automatically validates schema on startup
// It respects the SUWAYOMI_SCHEMA_VALIDATION environment variable
func AutoValidateSchema(client *Client, schemaPath string) error {
	mode := GetSchemaCompatibilityMode()

	if mode == CompatibilityIgnore {
		log.Println("⏭️  Schema validation disabled (SUWAYOMI_SCHEMA_VALIDATION=ignore)")
		return nil
	}

	log.Printf("🔍 Schema validation mode: %v", modeString(mode))
	return ValidateWithMode(client, schemaPath, mode)
}

func modeString(mode SchemaCompatibilityMode) string {
	switch mode {
	case CompatibilityStrict:
		return "strict"
	case CompatibilityWarn:
		return "warn"
	case CompatibilityIgnore:
		return "ignore"
	default:
		return "unknown"
	}
}
