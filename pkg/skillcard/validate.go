package skillcard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/redhat-et/skillimage/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

var compiledSchemas = map[string]*jsonschema.Schema{}

// Matches reverse-domain annotation keys such as com.example.security.reviewed.
var annotationKeyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?\.[A-Za-z0-9][A-Za-z0-9._-]*(?:\.[A-Za-z0-9][A-Za-z0-9._-]*)*$`)

func init() {
	compileSchema(APIVersionV1Alpha1, "skillcard-v1.json", schemas.SkillCardV1)
	compileSchema(APIVersionV1Alpha2, "skillcard-v1alpha2.json", schemas.SkillCardV1Alpha2)
}

func compileSchema(apiVersion, name string, data []byte) {
	var schemaDoc any
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		panic(fmt.Sprintf("failed to unmarshal embedded schema: %v", err))
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(name, schemaDoc); err != nil {
		panic(fmt.Sprintf("failed to add schema resource: %v", err))
	}

	compiled, err := compiler.Compile(name)
	if err != nil {
		panic(fmt.Sprintf("failed to compile schema: %v", err))
	}
	compiledSchemas[apiVersion] = compiled
}

// ValidationError represents a single validation error with field path and message.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate validates a SkillCard against the JSON Schema and additional semantic rules.
// Returns a slice of ValidationError for any validation failures.
func Validate(sc *SkillCard) ([]ValidationError, error) {
	var errors []ValidationError

	// Marshal SkillCard to YAML, then unmarshal to map[string]any to preserve kebab-case keys
	var serialized bytes.Buffer
	if err := Serialize(sc, &serialized); err != nil {
		return nil, fmt.Errorf("marshaling skillcard for validation: %w", err)
	}
	yamlBytes := serialized.Bytes()

	var doc map[string]any
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return nil, fmt.Errorf("unmarshaling skillcard for validation: %w", err)
	}

	compiledSchema, ok := compiledSchemas[sc.APIVersion]
	if !ok {
		return []ValidationError{{Field: "apiVersion", Message: fmt.Sprintf("unsupported API version %q (supported: %s, %s)", sc.APIVersion, APIVersionV1Alpha1, APIVersionV1Alpha2)}}, nil
	}

	if err := compiledSchema.Validate(doc); err != nil {
		// Extract validation errors
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			errors = append(errors, collectValidationErrors(validationErr)...)
		} else {
			return nil, fmt.Errorf("unexpected validation error type: %w", err)
		}
	}

	// Additional semver validation
	if sc.Metadata.Version != "" {
		version, err := semver.StrictNewVersion(sc.Metadata.Version)
		if err != nil {
			errors = append(errors, ValidationError{
				Field:   "metadata.version",
				Message: fmt.Sprintf("invalid semantic version: %v", err),
			})
		} else if sc.APIVersion == APIVersionV1Alpha2 && (version.Prerelease() != "" || version.Metadata() != "") {
			errors = append(errors, ValidationError{Field: "metadata.version", Message: "must be a MAJOR.MINOR.PATCH base version without prerelease or build identifiers"})
		}
	}

	if sc.APIVersion == APIVersionV1Alpha2 {
		errors = append(errors, validateAlpha2Metadata(sc.Metadata)...)
	}

	return errors, nil
}

func validateAlpha2Metadata(metadata Metadata) []ValidationError {
	var errs []ValidationError
	urls := map[string]string{
		"metadata.url":           metadata.URL,
		"metadata.documentation": metadata.Documentation,
		"metadata.support":       metadata.Support,
		"metadata.changelog":     metadata.Changelog,
	}
	for field, value := range urls {
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		localHTTP := u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
		if err != nil || u.Host == "" || (u.Scheme != "https" && !localHTTP) {
			errs = append(errs, ValidationError{Field: field, Message: "must be an absolute HTTPS URL (HTTP is allowed only for localhost)"})
		}
	}
	for key := range metadata.Annotations {
		field := "metadata.annotations." + key
		if strings.HasPrefix(key, "org.opencontainers.") || strings.HasPrefix(key, "io.skillimage.") {
			errs = append(errs, ValidationError{Field: field, Message: "uses a reserved annotation namespace"})
			continue
		}
		if !annotationKeyPattern.MatchString(key) {
			errs = append(errs, ValidationError{Field: field, Message: "must use reverse-domain notation"})
		}
	}
	return errs
}

// collectValidationErrors recursively collects leaf validation errors from the jsonschema ValidationError tree.
func collectValidationErrors(ve *jsonschema.ValidationError) []ValidationError {
	var errors []ValidationError

	// If there are causes, recurse into them
	if len(ve.Causes) > 0 {
		for _, cause := range ve.Causes {
			errors = append(errors, collectValidationErrors(cause)...)
		}
		return errors
	}

	// This is a leaf error - extract field path and message
	fieldPath := strings.Join(ve.InstanceLocation, ".")
	if fieldPath == "" {
		fieldPath = "(root)"
	}

	message := ve.Error()
	// Clean up the error message to remove the instance location prefix
	// The error format is typically: "instanceLocation: message"
	if idx := strings.Index(message, ": "); idx != -1 {
		message = message[idx+2:]
	}

	errors = append(errors, ValidationError{
		Field:   fieldPath,
		Message: message,
	})

	return errors
}
