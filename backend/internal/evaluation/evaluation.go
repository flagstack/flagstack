package evaluation

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

const BucketScale = 100_000

type MatchMode string

type Operator string

const (
	MatchAll MatchMode = "all"
	MatchAny MatchMode = "any"

	OperatorEquals                   Operator = "equals"
	OperatorNotEquals                Operator = "not_equals"
	OperatorIn                       Operator = "in"
	OperatorNotIn                    Operator = "not_in"
	OperatorContains                 Operator = "contains"
	OperatorNotContains              Operator = "not_contains"
	OperatorStartsWith               Operator = "starts_with"
	OperatorEndsWith                 Operator = "ends_with"
	OperatorGreaterThan              Operator = "greater_than"
	OperatorGreaterThanOrEqual       Operator = "greater_than_or_equal"
	OperatorLessThan                 Operator = "less_than"
	OperatorLessThanOrEqual          Operator = "less_than_or_equal"
	OperatorExists                   Operator = "exists"
	OperatorNotExists                Operator = "not_exists"
	OperatorMatchesRegex             Operator = "matches_regex"
	OperatorSemverGreaterThan        Operator = "semver_greater_than"
	OperatorSemverGreaterThanOrEqual Operator = "semver_greater_than_or_equal"
	OperatorSemverLessThan           Operator = "semver_less_than"
	OperatorSemverLessThanOrEqual    Operator = "semver_less_than_or_equal"
	OperatorInSegment                Operator = "in_segment"
	OperatorNotInSegment             Operator = "not_in_segment"
)

const (
	ReasonStatic         = "STATIC"
	ReasonDefault        = "DEFAULT"
	ReasonTargetingMatch = "TARGETING_MATCH"
	ReasonSplit          = "SPLIT"
	ReasonDisabled       = "DISABLED"
	ReasonError          = "ERROR"

	ErrorParse               = "PARSE_ERROR"
	ErrorTargetingKeyMissing = "TARGETING_KEY_MISSING"
	ErrorInvalidContext      = "INVALID_CONTEXT"
)

type Context struct {
	TargetingKey string
	Attributes   map[string]any
}

type Variant struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type Condition struct {
	Attribute string          `json:"attribute,omitempty"`
	Operator  Operator        `json:"operator"`
	Value     json.RawMessage `json:"value,omitempty"`
}

type Allocation struct {
	Variant string `json:"variant"`
	Weight  int    `json:"weight"`
}

type Outcome struct {
	Variant  string       `json:"variant,omitempty"`
	Rollout  []Allocation `json:"rollout,omitempty"`
	BucketBy string       `json:"bucket_by,omitempty"`
}

type Rule struct {
	ID         string      `json:"id"`
	Name       string      `json:"name,omitempty"`
	Match      MatchMode   `json:"match"`
	Conditions []Condition `json:"conditions"`
	Outcome    Outcome     `json:"outcome"`
}

type Policy struct {
	Rules       []Rule  `json:"rules,omitempty"`
	Fallthrough Outcome `json:"fallthrough,omitempty"`
}

type Segment struct {
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Match      MatchMode   `json:"match"`
	Conditions []Condition `json:"conditions"`
}

type Flag struct {
	ID            string
	EnvironmentID string
	Key           string
	Kind          string
	DefaultValue  json.RawMessage
	Enabled       bool
	Variants      []Variant
	Policy        Policy
}

type Result struct {
	Value        json.RawMessage `json:"value"`
	Variant      string          `json:"variant,omitempty"`
	Reason       string          `json:"reason"`
	RuleID       string          `json:"rule_id,omitempty"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

type evaluationError struct {
	code string
	err  error
}

func (e *evaluationError) Error() string { return e.err.Error() }
func (e *evaluationError) Unwrap() error { return e.err }

func Evaluate(flag Flag, ctx Context, segments []Segment) Result {
	if err := ValidateFlag(flag); err != nil {
		return errorResult(flag, ErrorParse, err)
	}

	segmentIndex := make(map[string]Segment, len(segments))
	for _, segment := range segments {
		if err := ValidateSegment(segment); err != nil {
			return errorResult(flag, ErrorParse, err)
		}
		segmentIndex[segment.Key] = segment
	}

	if !flag.Enabled {
		return Result{Value: cloneRaw(flag.DefaultValue), Variant: "default", Reason: ReasonDisabled}
	}

	for _, rule := range flag.Policy.Rules {
		matched, err := matchConditions(rule.Match, rule.Conditions, ctx, segmentIndex, map[string]bool{})
		if err != nil {
			return errorResultFromEvaluation(flag, err)
		}
		if !matched {
			continue
		}

		result, err := resolveOutcome(flag, rule.Outcome, ctx)
		if err != nil {
			return errorResultFromEvaluation(flag, err)
		}
		result.RuleID = rule.ID
		if len(rule.Outcome.Rollout) > 0 {
			result.Reason = ReasonSplit
		} else {
			result.Reason = ReasonTargetingMatch
		}
		return result
	}

	if outcomeEmpty(flag.Policy.Fallthrough) {
		if flag.Kind == "boolean" {
			return Result{Value: json.RawMessage("true"), Variant: "on", Reason: ReasonStatic}
		}
		return Result{Value: cloneRaw(flag.DefaultValue), Variant: "default", Reason: ReasonDefault}
	}

	result, err := resolveOutcome(flag, flag.Policy.Fallthrough, ctx)
	if err != nil {
		return errorResultFromEvaluation(flag, err)
	}
	if len(flag.Policy.Fallthrough.Rollout) > 0 {
		result.Reason = ReasonSplit
	} else {
		result.Reason = ReasonStatic
	}
	return result
}

func ValidateFlag(flag Flag) error {
	if strings.TrimSpace(flag.ID) == "" || strings.TrimSpace(flag.EnvironmentID) == "" {
		return fmt.Errorf("flag and environment IDs are required")
	}
	if !validKind(flag.Kind) {
		return fmt.Errorf("unsupported flag kind %q", flag.Kind)
	}
	if err := validateValueKind(flag.Kind, flag.DefaultValue); err != nil {
		return fmt.Errorf("default value: %w", err)
	}

	allowed := map[string]struct{}{"default": {}}
	if flag.Kind == "boolean" {
		allowed["on"] = struct{}{}
		allowed["off"] = struct{}{}
	}
	for _, variant := range flag.Variants {
		key := strings.TrimSpace(variant.Key)
		if key == "" {
			return fmt.Errorf("variant key is required")
		}
		if _, exists := allowed[key]; exists {
			return fmt.Errorf("variant key %q is reserved or duplicated", key)
		}
		if err := validateValueKind(flag.Kind, variant.Value); err != nil {
			return fmt.Errorf("variant %q: %w", key, err)
		}
		allowed[key] = struct{}{}
	}

	seenRules := make(map[string]struct{}, len(flag.Policy.Rules))
	for _, rule := range flag.Policy.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("rule ID is required")
		}
		if _, exists := seenRules[rule.ID]; exists {
			return fmt.Errorf("duplicate rule ID %q", rule.ID)
		}
		seenRules[rule.ID] = struct{}{}
		if err := validateMatchMode(rule.Match); err != nil {
			return fmt.Errorf("rule %q: %w", rule.ID, err)
		}
		if len(rule.Conditions) == 0 {
			return fmt.Errorf("rule %q must contain at least one condition", rule.ID)
		}
		for _, condition := range rule.Conditions {
			if err := validateCondition(condition); err != nil {
				return fmt.Errorf("rule %q: %w", rule.ID, err)
			}
		}
		if err := validateOutcome(rule.Outcome, allowed, true); err != nil {
			return fmt.Errorf("rule %q: %w", rule.ID, err)
		}
	}
	if err := validateOutcome(flag.Policy.Fallthrough, allowed, false); err != nil {
		return fmt.Errorf("fallthrough: %w", err)
	}
	return nil
}

func ValidateSegment(segment Segment) error {
	if strings.TrimSpace(segment.Key) == "" {
		return fmt.Errorf("segment key is required")
	}
	if err := validateMatchMode(segment.Match); err != nil {
		return fmt.Errorf("segment %q: %w", segment.Key, err)
	}
	if len(segment.Conditions) == 0 {
		return fmt.Errorf("segment %q must contain at least one condition", segment.Key)
	}
	for _, condition := range segment.Conditions {
		if err := validateCondition(condition); err != nil {
			return fmt.Errorf("segment %q: %w", segment.Key, err)
		}
	}
	return nil
}

func Bucket(environmentID, flagID, bucketValue string) int {
	input := "flagstack-v1\x00" + environmentID + "\x00" + flagID + "\x00" + bucketValue
	digest := sha256.Sum256([]byte(input))
	return int(binary.BigEndian.Uint32(digest[:4]) % BucketScale)
}

func resolveOutcome(flag Flag, outcome Outcome, ctx Context) (Result, error) {
	if outcome.Variant != "" {
		value, err := variantValue(flag, outcome.Variant)
		if err != nil {
			return Result{}, err
		}
		return Result{Value: value, Variant: outcome.Variant}, nil
	}

	bucketValue, err := bucketValue(ctx, outcome.BucketBy)
	if err != nil {
		return Result{}, err
	}
	bucket := Bucket(flag.EnvironmentID, flag.ID, bucketValue)
	cumulative := 0
	for _, allocation := range outcome.Rollout {
		cumulative += allocation.Weight
		if bucket < cumulative {
			value, err := variantValue(flag, allocation.Variant)
			if err != nil {
				return Result{}, err
			}
			return Result{Value: value, Variant: allocation.Variant}, nil
		}
	}
	return Result{}, &evaluationError{code: ErrorParse, err: fmt.Errorf("rollout did not resolve a variant")}
}

func variantValue(flag Flag, key string) (json.RawMessage, error) {
	switch key {
	case "default":
		return cloneRaw(flag.DefaultValue), nil
	case "on":
		if flag.Kind == "boolean" {
			return json.RawMessage("true"), nil
		}
	case "off":
		if flag.Kind == "boolean" {
			return json.RawMessage("false"), nil
		}
	}
	for _, variant := range flag.Variants {
		if variant.Key == key {
			return cloneRaw(variant.Value), nil
		}
	}
	return nil, &evaluationError{code: ErrorParse, err: fmt.Errorf("unknown variant %q", key)}
}

func bucketValue(ctx Context, bucketBy string) (string, error) {
	if bucketBy == "" || bucketBy == "targetingKey" {
		if ctx.TargetingKey == "" {
			return "", &evaluationError{code: ErrorTargetingKeyMissing, err: fmt.Errorf("targeting key is required for percentage rollout")}
		}
		return ctx.TargetingKey, nil
	}

	value, exists := contextValue(ctx, bucketBy)
	if !exists {
		return "", &evaluationError{code: ErrorInvalidContext, err: fmt.Errorf("bucket attribute %q is missing", bucketBy)}
	}
	encoded, err := scalarBucketValue(value)
	if err != nil {
		return "", &evaluationError{code: ErrorInvalidContext, err: fmt.Errorf("bucket attribute %q: %w", bucketBy, err)}
	}
	return encoded, nil
}

func scalarBucketValue(value any) (string, error) {
	switch value.(type) {
	case string, bool, json.Number, float64, float32,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	default:
		return "", fmt.Errorf("must be a scalar string, boolean or number")
	}
}

func matchConditions(mode MatchMode, conditions []Condition, ctx Context, segments map[string]Segment, visiting map[string]bool) (bool, error) {
	if mode == MatchAny {
		for _, condition := range conditions {
			matched, err := conditionMatches(condition, ctx, segments, visiting)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}

	for _, condition := range conditions {
		matched, err := conditionMatches(condition, ctx, segments, visiting)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func conditionMatches(condition Condition, ctx Context, segments map[string]Segment, visiting map[string]bool) (bool, error) {
	if condition.Operator == OperatorInSegment || condition.Operator == OperatorNotInSegment {
		value, err := decodeRaw(condition.Value)
		if err != nil {
			return false, &evaluationError{code: ErrorParse, err: err}
		}
		segmentKey, ok := value.(string)
		if !ok {
			return false, &evaluationError{code: ErrorParse, err: fmt.Errorf("segment condition must reference a string key")}
		}
		matched, err := matchSegment(segmentKey, ctx, segments, visiting)
		if err != nil {
			return false, err
		}
		if condition.Operator == OperatorNotInSegment {
			return !matched, nil
		}
		return matched, nil
	}

	actual, exists := contextValue(ctx, condition.Attribute)
	if condition.Operator == OperatorExists {
		return exists, nil
	}
	if condition.Operator == OperatorNotExists {
		return !exists, nil
	}
	if !exists {
		return false, nil
	}

	expected, err := decodeRaw(condition.Value)
	if err != nil {
		return false, &evaluationError{code: ErrorParse, err: err}
	}

	switch condition.Operator {
	case OperatorEquals:
		return equalValues(actual, expected), nil
	case OperatorNotEquals:
		return !equalValues(actual, expected), nil
	case OperatorIn, OperatorNotIn:
		values, ok := expected.([]any)
		if !ok {
			return false, &evaluationError{code: ErrorParse, err: fmt.Errorf("%s expects an array", condition.Operator)}
		}
		matched := false
		for _, candidate := range values {
			if equalValues(actual, candidate) {
				matched = true
				break
			}
		}
		if condition.Operator == OperatorNotIn {
			return !matched, nil
		}
		return matched, nil
	case OperatorContains, OperatorNotContains:
		matched := containsValue(actual, expected)
		if condition.Operator == OperatorNotContains {
			return !matched, nil
		}
		return matched, nil
	case OperatorStartsWith, OperatorEndsWith:
		actualString, actualOK := actual.(string)
		expectedString, expectedOK := expected.(string)
		if !actualOK || !expectedOK {
			return false, nil
		}
		if condition.Operator == OperatorStartsWith {
			return strings.HasPrefix(actualString, expectedString), nil
		}
		return strings.HasSuffix(actualString, expectedString), nil
	case OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorLessThan, OperatorLessThanOrEqual:
		actualNumber, actualOK := numberValue(actual)
		expectedNumber, expectedOK := numberValue(expected)
		if !actualOK || !expectedOK {
			return false, nil
		}
		switch condition.Operator {
		case OperatorGreaterThan:
			return actualNumber > expectedNumber, nil
		case OperatorGreaterThanOrEqual:
			return actualNumber >= expectedNumber, nil
		case OperatorLessThan:
			return actualNumber < expectedNumber, nil
		default:
			return actualNumber <= expectedNumber, nil
		}
	case OperatorMatchesRegex:
		actualString, actualOK := actual.(string)
		pattern, expectedOK := expected.(string)
		if !actualOK || !expectedOK {
			return false, nil
		}
		matched, err := regexp.MatchString(pattern, actualString)
		if err != nil {
			return false, &evaluationError{code: ErrorParse, err: err}
		}
		return matched, nil
	case OperatorSemverGreaterThan, OperatorSemverGreaterThanOrEqual, OperatorSemverLessThan, OperatorSemverLessThanOrEqual:
		actualString, actualOK := actual.(string)
		expectedString, expectedOK := expected.(string)
		if !actualOK || !expectedOK {
			return false, nil
		}
		comparison, ok := compareSemver(actualString, expectedString)
		if !ok {
			return false, nil
		}
		switch condition.Operator {
		case OperatorSemverGreaterThan:
			return comparison > 0, nil
		case OperatorSemverGreaterThanOrEqual:
			return comparison >= 0, nil
		case OperatorSemverLessThan:
			return comparison < 0, nil
		default:
			return comparison <= 0, nil
		}
	default:
		return false, &evaluationError{code: ErrorParse, err: fmt.Errorf("unsupported operator %q", condition.Operator)}
	}
}

func matchSegment(key string, ctx Context, segments map[string]Segment, visiting map[string]bool) (bool, error) {
	segment, exists := segments[key]
	if !exists {
		return false, nil
	}
	if visiting[key] {
		return false, &evaluationError{code: ErrorParse, err: fmt.Errorf("segment cycle detected at %q", key)}
	}
	visiting[key] = true
	defer delete(visiting, key)
	return matchConditions(segment.Match, segment.Conditions, ctx, segments, visiting)
}

func contextValue(ctx Context, path string) (any, bool) {
	if path == "targetingKey" {
		if ctx.TargetingKey == "" {
			return nil, false
		}
		return ctx.TargetingKey, true
	}
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	var current any = ctx.Attributes
	for _, part := range parts {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = mapping[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func containsValue(actual, expected any) bool {
	switch value := actual.(type) {
	case string:
		expectedString, ok := expected.(string)
		return ok && strings.Contains(value, expectedString)
	case []any:
		for _, candidate := range value {
			if equalValues(candidate, expected) {
				return true
			}
		}
	case []string:
		expectedString, ok := expected.(string)
		if !ok {
			return false
		}
		for _, candidate := range value {
			if candidate == expectedString {
				return true
			}
		}
	case map[string]any:
		expectedString, ok := expected.(string)
		if !ok {
			return false
		}
		_, exists := value[expectedString]
		return exists
	}
	return false
}

func equalValues(left, right any) bool {
	leftNumber, leftOK := numberValue(left)
	rightNumber, rightOK := numberValue(right)
	if leftOK && rightOK {
		return leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}

func numberValue(value any) (float64, bool) {
	switch number := value.(type) {
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int8:
		return float64(number), true
	case int16:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint8:
		return float64(number), true
	case uint16:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func compareSemver(left, right string) (int, bool) {
	left = normalizeSemver(left)
	right = normalizeSemver(right)
	if !semver.IsValid(left) || !semver.IsValid(right) {
		return 0, false
	}
	return semver.Compare(left, right), true
}

func normalizeSemver(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && !strings.HasPrefix(value, "v") {
		return "v" + value
	}
	return value
}

func validateOutcome(outcome Outcome, allowed map[string]struct{}, required bool) error {
	hasVariant := strings.TrimSpace(outcome.Variant) != ""
	hasRollout := len(outcome.Rollout) > 0
	if hasVariant && hasRollout {
		return fmt.Errorf("outcome cannot contain both a variant and a rollout")
	}
	if !hasVariant && !hasRollout {
		if required {
			return fmt.Errorf("outcome must contain a variant or rollout")
		}
		return nil
	}
	if hasVariant {
		if _, exists := allowed[outcome.Variant]; !exists {
			return fmt.Errorf("unknown variant %q", outcome.Variant)
		}
		return nil
	}

	total := 0
	for _, allocation := range outcome.Rollout {
		if _, exists := allowed[allocation.Variant]; !exists {
			return fmt.Errorf("unknown rollout variant %q", allocation.Variant)
		}
		if allocation.Weight <= 0 {
			return fmt.Errorf("rollout weights must be positive")
		}
		total += allocation.Weight
	}
	if total != BucketScale {
		return fmt.Errorf("rollout weights must total %d", BucketScale)
	}
	return nil
}

func validateCondition(condition Condition) error {
	switch condition.Operator {
	case OperatorInSegment, OperatorNotInSegment:
		value, err := decodeRaw(condition.Value)
		if err != nil {
			return fmt.Errorf("segment reference: %w", err)
		}
		if segmentKey, ok := value.(string); !ok || strings.TrimSpace(segmentKey) == "" {
			return fmt.Errorf("segment reference must be a non-empty string")
		}
		return nil
	case OperatorExists, OperatorNotExists:
		if strings.TrimSpace(condition.Attribute) == "" {
			return fmt.Errorf("condition attribute is required")
		}
		return nil
	}

	if strings.TrimSpace(condition.Attribute) == "" {
		return fmt.Errorf("condition attribute is required")
	}
	value, err := decodeRaw(condition.Value)
	if err != nil {
		return fmt.Errorf("condition value: %w", err)
	}

	switch condition.Operator {
	case OperatorEquals, OperatorNotEquals,
		OperatorContains, OperatorNotContains,
		OperatorStartsWith, OperatorEndsWith,
		OperatorGreaterThan, OperatorGreaterThanOrEqual,
		OperatorLessThan, OperatorLessThanOrEqual:
		return nil
	case OperatorIn, OperatorNotIn:
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("%s condition value must be an array", condition.Operator)
		}
		return nil
	case OperatorMatchesRegex:
		pattern, ok := value.(string)
		if !ok {
			return fmt.Errorf("regex condition value must be a string")
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regular expression: %w", err)
		}
		return nil
	case OperatorSemverGreaterThan, OperatorSemverGreaterThanOrEqual, OperatorSemverLessThan, OperatorSemverLessThanOrEqual:
		version, ok := value.(string)
		if !ok || !semver.IsValid(normalizeSemver(version)) {
			return fmt.Errorf("semantic-version condition value must be a valid semantic version")
		}
		return nil
	default:
		return fmt.Errorf("unsupported operator %q", condition.Operator)
	}
}

func validateMatchMode(mode MatchMode) error {
	if mode != MatchAll && mode != MatchAny {
		return fmt.Errorf("match mode must be %q or %q", MatchAll, MatchAny)
	}
	return nil
}

func validateValueKind(kind string, raw json.RawMessage) error {
	value, err := decodeRaw(raw)
	if err != nil {
		return err
	}
	switch kind {
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}
	case "number":
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("must be a number")
		}
	case "json":
		return nil
	default:
		return fmt.Errorf("unsupported flag kind %q", kind)
	}
	return nil
}

func decodeRaw(raw json.RawMessage) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("JSON value is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("multiple JSON values are not allowed")
	}
	return value, nil
}

func validKind(kind string) bool {
	return kind == "boolean" || kind == "string" || kind == "number" || kind == "json"
}

func outcomeEmpty(outcome Outcome) bool {
	return strings.TrimSpace(outcome.Variant) == "" && len(outcome.Rollout) == 0
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), raw...)
}

func errorResult(flag Flag, code string, err error) Result {
	return Result{
		Value:        cloneRaw(flag.DefaultValue),
		Variant:      "default",
		Reason:       ReasonError,
		ErrorCode:    code,
		ErrorMessage: err.Error(),
	}
}

func errorResultFromEvaluation(flag Flag, err error) Result {
	if evaluationErr, ok := err.(*evaluationError); ok {
		return errorResult(flag, evaluationErr.code, evaluationErr.err)
	}
	return errorResult(flag, ErrorParse, err)
}
