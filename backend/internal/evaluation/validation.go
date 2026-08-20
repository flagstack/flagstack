package evaluation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateDefinition validates a flag's variant and policy configuration without
// requiring a persisted flag or environment ID.
func ValidateDefinition(kind string, defaultValue json.RawMessage, variants []Variant, policy Policy) error {
	if err := ValidateFlag(Flag{
		ID:            "validation-flag",
		EnvironmentID: "validation-environment",
		Kind:          kind,
		DefaultValue:  defaultValue,
		Variants:      variants,
		Policy:        policy,
	}); err != nil {
		return err
	}
	return validatePortablePolicy(policy)
}

// ValidateSegmentSet validates segment syntax, unique keys, references and
// recursive dependencies before a project configuration is persisted.
func ValidateSegmentSet(segments []Segment) error {
	index := make(map[string]Segment, len(segments))
	for _, segment := range segments {
		if err := ValidateSegment(segment); err != nil {
			return err
		}
		if err := validatePortableConditions(segment.Conditions); err != nil {
			return fmt.Errorf("segment %q: %w", segment.Key, err)
		}
		if _, exists := index[segment.Key]; exists {
			return fmt.Errorf("duplicate segment key %q", segment.Key)
		}
		index[segment.Key] = segment
	}

	for _, segment := range segments {
		for _, referenced := range segmentReferences(segment.Conditions) {
			if _, exists := index[referenced]; !exists {
				return fmt.Errorf("segment %q references unknown segment %q", segment.Key, referenced)
			}
	}

	visiting := make(map[string]bool, len(index))
	visited := make(map[string]bool, len(index))
	var visit func(string) error
	visit = func(key string) error {
		if visited[key] {
			return nil
		}
		if visiting[key] {
			return fmt.Errorf("segment cycle detected at %q", key)
		}
		visiting[key] = true
		for _, referenced := range segmentReferences(index[key].Conditions) {
			if err := visit(referenced); err != nil {
				return err
			}
		}
		delete(visiting, key)
		visited[key] = true
		return nil
	}

	for key := range index {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePolicySegments ensures every segment referenced by a policy exists.
func ValidatePolicySegments(policy Policy, segments []Segment) error {
	if err := validatePortablePolicy(policy); err != nil {
		return err
	}
	known := make(map[string]struct{}, len(segments))
	for _, segment := range segments {
		known[segment.Key] = struct{}{}
	}
	for _, rule := range policy.Rules {
		for _, referenced := range segmentReferences(rule.Conditions) {
			if _, exists := known[referenced]; !exists {
				return fmt.Errorf("rule %q references unknown segment %q", rule.ID, referenced)
			}
		}
	}
	return nil
}

func validatePortablePolicy(policy Policy) error {
	for _, rule := range policy.Rules {
		if err := validatePortableConditions(rule.Conditions); err != nil {
			return fmt.Errorf("rule %q: %w", rule.ID, err)
		}
	}
	return nil
}

func validatePortableConditions(conditions []Condition) error {
	for _, condition := range conditions {
		if condition.Operator != OperatorMatchesRegex {
			continue
		}
		value, err := decodeRaw(condition.Value)
		if err != nil {
			return err
		}
		pattern, ok := value.(string)
		if !ok {
			return fmt.Errorf("regex condition value must be a string")
		}
		if err := ValidateRegexPattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func segmentReferences(conditions []Condition) []string {
	references := make([]string, 0)
	for _, condition := range conditions {
		if condition.Operator != OperatorInSegment && condition.Operator != OperatorNotInSegment {
			continue
		}
		value, err := decodeRaw(condition.Value)
		if err != nil {
			continue
		}
		key, ok := value.(string)
		if ok && strings.TrimSpace(key) != "" {
			references = append(references, key)
		}
	}
	return references
}
