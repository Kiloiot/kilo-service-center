package passwordpolicy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testLetter            = "a"
	testDigit             = "1"
	testSymbol            = "!"
	testLengthOnlyMin     = 4
	testLengthOnlyMax     = 6
	testRulesDescription  = "password must be 8 to 128 characters long and contain a letter and a digit"
	testLengthDescription = "password must be 4 to 6 characters long"
	testLetterDescription = "password must be 4 to 6 characters long and contain a letter"
	testDigitDescription  = "password must be 4 to 6 characters long and contain a digit"
)

func TestDescribe_StatesEveryRuleThePolicyHolds(t *testing.T) {
	lengthOnly := Policy{MinLength: testLengthOnlyMin, MaxLength: testLengthOnlyMax}
	letter, digit := lengthOnly, lengthOnly
	letter.RequiresLetter = Rules.RequiresLetter
	digit.RequiresDigit = Rules.RequiresDigit

	assert.Equal(t, testRulesDescription, Rules.Describe())
	assert.Equal(t, testLengthDescription, lengthOnly.Describe())
	assert.Equal(t, testLetterDescription, letter.Describe())
	assert.Equal(t, testDigitDescription, digit.Describe())
}

func TestAccepts_EnforcesTheRulesDescribeStates(t *testing.T) {
	shortest := strings.Repeat(testLetter, int(Rules.MinLength)-1) + testDigit
	longest := strings.Repeat(testLetter, int(Rules.MaxLength)-1) + testDigit

	assert.True(t, Rules.Accepts(shortest))
	assert.True(t, Rules.Accepts(longest))
	assert.False(t, Rules.Accepts(shortest[1:]))
	assert.False(t, Rules.Accepts(longest+testDigit))
	assert.False(t, Rules.Accepts(strings.Repeat(testLetter, int(Rules.MinLength))))
	assert.False(t, Rules.Accepts(strings.Repeat(testDigit, int(Rules.MinLength))))
	assert.True(t, Policy{MinLength: testLengthOnlyMin, MaxLength: testLengthOnlyMax}.Accepts(strings.Repeat(testSymbol, testLengthOnlyMin)))
}
