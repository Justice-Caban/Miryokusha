package suwayomi

import (
	"fmt"
	"strings"
)

// SchemaValidationResult contains the results of schema validation
type SchemaValidationResult struct {
	IsValid          bool
	Errors           []string
	Warnings         []string
	MissingTypes     []string
	MissingFields    []string
	TypeMismatches   []string
	DeprecatedFields []string
}

// SchemaValidator validates the actual schema against expected structure
type SchemaValidator struct {
	expected *IntrospectionResult
	actual   *IntrospectionResult
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(expected, actual *IntrospectionResult) *SchemaValidator {
	return &SchemaValidator{
		expected: expected,
		actual:   actual,
	}
}

// Validate performs comprehensive schema validation
func (sv *SchemaValidator) Validate() *SchemaValidationResult {
	result := &SchemaValidationResult{
		IsValid:          true,
		Errors:           []string{},
		Warnings:         []string{},
		MissingTypes:     []string{},
		MissingFields:    []string{},
		TypeMismatches:   []string{},
		DeprecatedFields: []string{},
	}

	// Check root types
	sv.validateRootTypes(result)

	// Check each expected type exists
	for _, expectedType := range sv.expected.Schema.Types {
		// Skip internal GraphQL types
		if strings.HasPrefix(expectedType.Name, "__") {
			continue
		}

		actualType := sv.actual.GetTypeByName(expectedType.Name)
		if actualType == nil {
			result.MissingTypes = append(result.MissingTypes, expectedType.Name)
			result.Errors = append(result.Errors,
				fmt.Sprintf("Type '%s' is missing from server schema", expectedType.Name))
			result.IsValid = false
			continue
		}

		// Validate type kind matches
		if expectedType.Kind != actualType.Kind {
			result.TypeMismatches = append(result.TypeMismatches,
				fmt.Sprintf("%s: expected %s, got %s", expectedType.Name, expectedType.Kind, actualType.Kind))
			result.Errors = append(result.Errors,
				fmt.Sprintf("Type '%s' kind mismatch: expected %s, got %s",
					expectedType.Name, expectedType.Kind, actualType.Kind))
			result.IsValid = false
			continue
		}

		// Validate fields for object/interface types
		if expectedType.Kind == "OBJECT" || expectedType.Kind == "INTERFACE" {
			sv.validateFields(&expectedType, actualType, result)
		}

		// Validate enum values
		if expectedType.Kind == "ENUM" {
			sv.validateEnumValues(&expectedType, actualType, result)
		}

		// Validate input fields
		if expectedType.Kind == "INPUT_OBJECT" {
			sv.validateInputFields(&expectedType, actualType, result)
		}
	}

	return result
}

func (sv *SchemaValidator) validateRootTypes(result *SchemaValidationResult) {
	// Check Query type
	if sv.expected.Schema.QueryType != nil {
		if sv.actual.Schema.QueryType == nil {
			result.Errors = append(result.Errors, "Query type is missing from server schema")
			result.IsValid = false
		} else if sv.expected.Schema.QueryType.Name != sv.actual.Schema.QueryType.Name {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Query type name mismatch: expected %s, got %s",
					sv.expected.Schema.QueryType.Name, sv.actual.Schema.QueryType.Name))
			result.IsValid = false
		}
	}

	// Check Mutation type
	if sv.expected.Schema.MutationType != nil {
		if sv.actual.Schema.MutationType == nil {
			result.Warnings = append(result.Warnings, "Mutation type is missing from server schema")
		} else if sv.expected.Schema.MutationType.Name != sv.actual.Schema.MutationType.Name {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Mutation type name mismatch: expected %s, got %s",
					sv.expected.Schema.MutationType.Name, sv.actual.Schema.MutationType.Name))
		}
	}

	// Check Subscription type
	if sv.expected.Schema.SubscriptionType != nil {
		if sv.actual.Schema.SubscriptionType == nil {
			result.Warnings = append(result.Warnings, "Subscription type is missing from server schema")
		} else if sv.expected.Schema.SubscriptionType.Name != sv.actual.Schema.SubscriptionType.Name {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Subscription type name mismatch: expected %s, got %s",
					sv.expected.Schema.SubscriptionType.Name, sv.actual.Schema.SubscriptionType.Name))
		}
	}
}

func (sv *SchemaValidator) validateFields(expectedType, actualType *SchemaType, result *SchemaValidationResult) {
	for _, expectedField := range expectedType.Fields {
		actualField := actualType.GetFieldByName(expectedField.Name)
		if actualField == nil {
			fieldPath := fmt.Sprintf("%s.%s", expectedType.Name, expectedField.Name)
			result.MissingFields = append(result.MissingFields, fieldPath)
			result.Errors = append(result.Errors,
				fmt.Sprintf("Field '%s' is missing from type '%s'", expectedField.Name, expectedType.Name))
			result.IsValid = false
			continue
		}

		// Check if field type matches
		expectedTypeStr := sv.typeRefToString(expectedField.Type)
		actualTypeStr := sv.typeRefToString(actualField.Type)
		if expectedTypeStr != actualTypeStr {
			result.TypeMismatches = append(result.TypeMismatches,
				fmt.Sprintf("%s.%s: expected %s, got %s",
					expectedType.Name, expectedField.Name, expectedTypeStr, actualTypeStr))
			result.Errors = append(result.Errors,
				fmt.Sprintf("Field '%s.%s' type mismatch: expected %s, got %s",
					expectedType.Name, expectedField.Name, expectedTypeStr, actualTypeStr))
			result.IsValid = false
		}

		// Check for deprecation
		if actualField.IsDeprecated && !expectedField.IsDeprecated {
			result.DeprecatedFields = append(result.DeprecatedFields,
				fmt.Sprintf("%s.%s", expectedType.Name, expectedField.Name))
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Field '%s.%s' is deprecated: %s",
					expectedType.Name, expectedField.Name, actualField.DeprecationReason))
		}

		// Validate arguments
		sv.validateFieldArgs(expectedType.Name, &expectedField, actualField, result)
	}
}

func (sv *SchemaValidator) validateFieldArgs(typeName string, expectedField, actualField *SchemaField, result *SchemaValidationResult) {
	for _, expectedArg := range expectedField.Args {
		found := false
		for _, actualArg := range actualField.Args {
			if expectedArg.Name == actualArg.Name {
				found = true
				// Check argument type matches
				expectedArgTypeStr := sv.typeRefToString(expectedArg.Type)
				actualArgTypeStr := sv.typeRefToString(actualArg.Type)
				if expectedArgTypeStr != actualArgTypeStr {
					result.TypeMismatches = append(result.TypeMismatches,
						fmt.Sprintf("%s.%s(%s): expected %s, got %s",
							typeName, expectedField.Name, expectedArg.Name,
							expectedArgTypeStr, actualArgTypeStr))
					result.Errors = append(result.Errors,
						fmt.Sprintf("Argument '%s' of field '%s.%s' type mismatch: expected %s, got %s",
							expectedArg.Name, typeName, expectedField.Name,
							expectedArgTypeStr, actualArgTypeStr))
					result.IsValid = false
				}
				break
			}
		}

		if !found {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Argument '%s' is missing from field '%s.%s'",
					expectedArg.Name, typeName, expectedField.Name))
			result.IsValid = false
		}
	}
}

func (sv *SchemaValidator) validateEnumValues(expectedType, actualType *SchemaType, result *SchemaValidationResult) {
	for _, expectedValue := range expectedType.EnumValues {
		found := false
		for _, actualValue := range actualType.EnumValues {
			if expectedValue.Name == actualValue.Name {
				found = true
				break
			}
		}

		if !found {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Enum value '%s' is missing from enum '%s'",
					expectedValue.Name, expectedType.Name))
			result.IsValid = false
		}
	}
}

func (sv *SchemaValidator) validateInputFields(expectedType, actualType *SchemaType, result *SchemaValidationResult) {
	for _, expectedField := range expectedType.InputFields {
		found := false
		for _, actualField := range actualType.InputFields {
			if expectedField.Name == actualField.Name {
				found = true
				// Check input field type matches
				expectedTypeStr := sv.typeRefToString(expectedField.Type)
				actualTypeStr := sv.typeRefToString(actualField.Type)
				if expectedTypeStr != actualTypeStr {
					result.TypeMismatches = append(result.TypeMismatches,
						fmt.Sprintf("%s.%s: expected %s, got %s",
							expectedType.Name, expectedField.Name, expectedTypeStr, actualTypeStr))
					result.Errors = append(result.Errors,
						fmt.Sprintf("Input field '%s.%s' type mismatch: expected %s, got %s",
							expectedType.Name, expectedField.Name, expectedTypeStr, actualTypeStr))
					result.IsValid = false
				}
				break
			}
		}

		if !found {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Input field '%s' is missing from input object '%s'",
					expectedField.Name, expectedType.Name))
			result.IsValid = false
		}
	}
}

func (sv *SchemaValidator) typeRefToString(typeRef SchemaTypeRef) string {
	switch typeRef.Kind {
	case "NON_NULL":
		return sv.typeRefToString(*typeRef.OfType) + "!"
	case "LIST":
		return "[" + sv.typeRefToString(*typeRef.OfType) + "]"
	default:
		return typeRef.Name
	}
}

// Report generates a human-readable validation report
func (result *SchemaValidationResult) Report() string {
	var report strings.Builder

	if result.IsValid {
		report.WriteString("✅ Schema Validation PASSED\n\n")
	} else {
		report.WriteString("❌ Schema Validation FAILED\n\n")
	}

	if len(result.Errors) > 0 {
		report.WriteString(fmt.Sprintf("Errors (%d):\n", len(result.Errors)))
		for _, err := range result.Errors {
			report.WriteString(fmt.Sprintf("  ❌ %s\n", err))
		}
		report.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		report.WriteString(fmt.Sprintf("Warnings (%d):\n", len(result.Warnings)))
		for _, warn := range result.Warnings {
			report.WriteString(fmt.Sprintf("  ⚠️  %s\n", warn))
		}
		report.WriteString("\n")
	}

	if len(result.MissingTypes) > 0 {
		report.WriteString(fmt.Sprintf("Missing Types (%d):\n", len(result.MissingTypes)))
		for _, typeName := range result.MissingTypes {
			report.WriteString(fmt.Sprintf("  • %s\n", typeName))
		}
		report.WriteString("\n")
	}

	if len(result.MissingFields) > 0 {
		report.WriteString(fmt.Sprintf("Missing Fields (%d):\n", len(result.MissingFields)))
		for _, fieldPath := range result.MissingFields {
			report.WriteString(fmt.Sprintf("  • %s\n", fieldPath))
		}
		report.WriteString("\n")
	}

	if len(result.TypeMismatches) > 0 {
		report.WriteString(fmt.Sprintf("Type Mismatches (%d):\n", len(result.TypeMismatches)))
		for _, mismatch := range result.TypeMismatches {
			report.WriteString(fmt.Sprintf("  • %s\n", mismatch))
		}
		report.WriteString("\n")
	}

	if len(result.DeprecatedFields) > 0 {
		report.WriteString(fmt.Sprintf("Deprecated Fields (%d):\n", len(result.DeprecatedFields)))
		for _, field := range result.DeprecatedFields {
			report.WriteString(fmt.Sprintf("  • %s\n", field))
		}
		report.WriteString("\n")
	}

	if result.IsValid && len(result.Warnings) == 0 {
		report.WriteString("All schema validations passed successfully!\n")
	}

	return report.String()
}

// ValidateClientSchema validates the client's expectations against server schema at runtime
func ValidateClientSchema(client *Client) (*SchemaValidationResult, error) {
	// Introspect the server schema
	actualSchema, err := client.IntrospectSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to introspect server schema: %w", err)
	}

	// Try to load expected schema from file
	expectedSchema, err := LoadSchemaFromFile("schema/suwayomi_schema.json")
	if err != nil {
		// If no expected schema file exists, we can't validate
		return &SchemaValidationResult{
			IsValid:  true,
			Warnings: []string{"No expected schema file found for validation"},
		}, nil
	}

	// Create validator and validate
	validator := NewSchemaValidator(expectedSchema, actualSchema)
	return validator.Validate(), nil
}
