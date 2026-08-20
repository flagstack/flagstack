package evaluation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type conformanceVectors struct {
	SchemaVersion       int                  `json:"schema_version"`
	BucketVectors       []bucketVector       `json:"bucket_vectors"`
	ScalarBucketVectors []scalarBucketVector `json:"scalar_bucket_vectors"`
	SemverVectors       []semverVector       `json:"semver_vectors"`
	RegexVectors        []regexVector        `json:"regex_vectors"`
}

type bucketVector struct {
	Name          string `json:"name"`
	EnvironmentID string `json:"environment_id"`
	FlagID        string `json:"flag_id"`
	BucketValue   string `json:"bucket_value"`
	Bucket        int    `json:"bucket"`
}

type scalarBucketVector struct {
	Name       string `json:"name"`
	Value      any    `json:"value"`
	Serialized string `json:"serialized"`
	Bucket     int    `json:"bucket"`
}

type semverVector struct {
	Left       string `json:"left"`
	Right      string `json:"right"`
	Comparison int    `json:"comparison"`
}

type regexVector struct {
	Pattern  string `json:"pattern"`
	Portable bool   `json:"portable"`
}

func TestEvaluationV1ConformanceVectors(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "spec", "evaluation-v1-vectors.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var vectors conformanceVectors
	if err := decoder.Decode(&vectors); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if vectors.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", vectors.SchemaVersion)
	}

	for _, vector := range vectors.BucketVectors {
		t.Run("bucket/"+vector.Name, func(t *testing.T) {
			if got := Bucket(vector.EnvironmentID, vector.FlagID, vector.BucketValue); got != vector.Bucket {
				t.Fatalf("Bucket() = %d, want %d", got, vector.Bucket)
			}
		})
	}

	for _, vector := range vectors.ScalarBucketVectors {
		t.Run("scalar/"+vector.Name, func(t *testing.T) {
			serialized, err := scalarBucketValue(vector.Value)
			if err != nil {
				t.Fatalf("scalarBucketValue() error = %v", err)
			}
			if serialized != vector.Serialized {
				t.Fatalf("scalarBucketValue() = %q, want %q", serialized, vector.Serialized)
			}
			if got := Bucket("env-1", "flag-1", serialized); got != vector.Bucket {
				t.Fatalf("Bucket() = %d, want %d", got, vector.Bucket)
			}
		})
	}

	for _, vector := range vectors.SemverVectors {
		t.Run("semver/"+vector.Left+"/"+vector.Right, func(t *testing.T) {
			comparison, ok := compareSemver(vector.Left, vector.Right)
			if !ok {
				t.Fatal("compareSemver() rejected a conformance vector")
			}
			if sign(comparison) != vector.Comparison {
				t.Fatalf("compareSemver() = %d, want sign %d", comparison, vector.Comparison)
			}
		})
	}

	for _, vector := range vectors.RegexVectors {
		t.Run("regex/"+vector.Pattern, func(t *testing.T) {
			err := ValidateRegexPattern(vector.Pattern)
			if vector.Portable && err != nil {
				t.Fatalf("ValidateRegexPattern() error = %v", err)
			}
			if !vector.Portable && err == nil {
				t.Fatal("ValidateRegexPattern() error = nil, want portability rejection")
			}
		})
	}
}

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
