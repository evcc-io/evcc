package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIValidation(t *testing.T) {
	// Get the path to the OpenAPI spec file
	openapiPath := filepath.Join("openapi.yaml")

	// Check if the file exists
	_, err := os.Stat(openapiPath)
	require.NoError(t, err, "OpenAPI file should exist at %s", openapiPath)

	// Load the OpenAPI specification
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(openapiPath)
	require.NoError(t, err, "Failed to load OpenAPI specification")

	// Validate the OpenAPI specification
	err = doc.Validate(loader.Context)
	assert.NoError(t, err, "OpenAPI specification should be valid")

	// Additional checks
	assert.NotNil(t, doc.Info, "OpenAPI spec should have info section")
	assert.NotEmpty(t, doc.Info.Title, "OpenAPI spec should have a title")
	assert.NotEmpty(t, doc.Info.Version, "OpenAPI spec should have a version")
	assert.NotEmpty(t, doc.Paths, "OpenAPI spec should have paths defined")
}
