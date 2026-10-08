// Package passwordpolicy holds the rules a new password must meet, the check
// that enforces them and the sentence that states them, so the three cannot
// disagree.
package passwordpolicy

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

const (
	describeLengthFmt   = "password must be %d to %d characters long"
	describeContains    = " and contain "
	describeJoin        = " and "
	describeLetterLabel = "a letter"
	describeDigitLabel  = "a digit"
)

// Policy states the rules a new password must meet; lengths count runes.
type Policy struct {
	MinLength      int32
	MaxLength      int32
	RequiresLetter bool
	RequiresDigit  bool
}

// Rules is the policy every new password must meet.
var Rules = Policy{
	MinLength:      config.AuthPasswordMinLength,
	MaxLength:      config.AuthPasswordMaxLength,
	RequiresLetter: true,
	RequiresDigit:  true,
}

// Accepts reports whether the password meets every rule of the policy.
func (p Policy) Accepts(password string) bool {
	length := utf8.RuneCountInString(password)
	if length < int(p.MinLength) || length > int(p.MaxLength) {
		return false
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		hasLetter = hasLetter || unicode.IsLetter(r)
		hasDigit = hasDigit || unicode.IsDigit(r)
	}
	return (hasLetter || !p.RequiresLetter) && (hasDigit || !p.RequiresDigit)
}

// Describe states the policy as one sentence, naming only the rules it holds.
func (p Policy) Describe() string {
	sentence := fmt.Sprintf(describeLengthFmt, p.MinLength, p.MaxLength)

	var requirements []string
	if p.RequiresLetter {
		requirements = append(requirements, describeLetterLabel)
	}
	if p.RequiresDigit {
		requirements = append(requirements, describeDigitLabel)
	}
	if len(requirements) == 0 {
		return sentence
	}
	return sentence + describeContains + strings.Join(requirements, describeJoin)
}
