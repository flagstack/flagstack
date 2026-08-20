package evaluation

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateRegexPattern validates the portable SwitchOnYourCode v1 regular-expression
// subset. SwitchOnYourCode uses RE2 semantics, but the portable subset is intentionally
// narrower where an officially supported SDK runtime cannot reproduce a valid
// RE2 construct safely.
func ValidateRegexPattern(pattern string) error {
	if strings.Contains(pattern, "[[:") {
		return fmt.Errorf("POSIX character classes are not supported by the SwitchOnYourCode portable regex subset")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid regular expression: %w", err)
	}
	return nil
}
