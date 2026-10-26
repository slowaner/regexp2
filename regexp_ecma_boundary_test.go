package regexp2

import (
	"testing"

	"github.com/dlclark/regexp2/v2/syntax"
)

// WordCharacters is shared by word escapes and boundary assertions in ECMA-262.
// Only Unicode ignore-case adds non-ASCII simple-fold equivalents of ASCII.
func TestECMAWordBoundarySemantics(t *testing.T) {
	for _, tt := range []struct {
		name, pattern, input string
		options              RegexOptions
		want                 bool
	}{
		{"Unicode Kelvin ignore case", `\bK\b`, "K", ECMAScript | Unicode | IgnoreCase, true},
		{"ASCII next to Unicode", `\ba\b`, "éaé", ECMAScript | Unicode, true},
		{"legacy ASCII next to Unicode", `\ba\b`, "éaé", ECMAScript, true},
		{"Greek is not a word", `\bα\b`, "α", ECMAScript | Unicode, false},
		{"nonboundary next to Unicode", `\Ba\B`, "éaé", ECMAScript | Unicode, false},
		{"Unicode nonword at both ends", `\Bé\B`, "é", ECMAScript | Unicode, true},
		{"combining marks", `\ba\b`, "\u0301a\u0301", ECMAScript | Unicode, true},
		{"Unicode decimal digits", `\ba\b`, "١a١", ECMAScript | Unicode, true},
		{"Unicode connector", `\ba\b`, "‿a‿", ECMAScript | Unicode, true},
		{"empty boundary", `\b`, "", ECMAScript | Unicode, false},
		{"empty nonboundary", `\B`, "", ECMAScript | Unicode, true},
		{"ASCII word", `\b_A9\b`, " _A9 ", ECMAScript | Unicode, true},
		{"Kelvin without ignore case", `\bK\b`, "K", ECMAScript | Unicode, false},
		{"legacy Kelvin ignore case", `\bK\b`, "K", ECMAScript | IgnoreCase, false},
		{"Unicode long s ignore case", `\bſ\b`, "ſ", ECMAScript | Unicode | IgnoreCase, true},
		{"dotted I remains nonword", `\Bİ\B`, "İ", ECMAScript | Unicode | IgnoreCase, true},
		{"scoped enable", `(?i:\bK\b)`, "K", ECMAScript | Unicode, true},
		{"scoped disable", `(?-i:\bK\b)`, "K", ECMAScript | Unicode | IgnoreCase, false},
		{"nested disable", `(?i:(?-i:\bK\b))`, "K", ECMAScript | Unicode, false},
		{"nested restore", `(?i:(?-i:x) \bK\b)`, "x K", ECMAScript | Unicode, true},
		{"outer restore", `(?i:x) \bK\b`, "x K", ECMAScript | Unicode, false},
		{"scoped nonboundary", `(?-i:\BK\B)`, "K", ECMAScript | Unicode | IgnoreCase, true},
		{"lookarounds", `(?<=\b)K(?=\b)`, "K", ECMAScript | Unicode | IgnoreCase, true},
		{"right to left ASCII", `\ba\b`, "éaé", ECMAScript | Unicode | RightToLeft, true},
		{"right to left scoped disable", `(?-i:\BK\B)`, "K", ECMAScript | Unicode | IgnoreCase | RightToLeft, true},
		{"default mode unchanged", `\bé\b`, "é", None, true},
		{"RE2 mode unchanged", `\bé\b`, "é", RE2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			re := MustCompile(tt.pattern, tt.options)

			got, err := re.MatchString(tt.input)

			if err != nil || got != tt.want {
				t.Fatalf("%s on %q options %v = %v, %v; want %v", tt.pattern, tt.input, tt.options, got, err, tt.want)
			}
		})
	}
}

func TestECMAWordBoundaryLandmarkFiltering(t *testing.T) {
	for _, tt := range []struct {
		name, pattern, input string
		options              RegexOptions
	}{
		{"ASCII beside Unicode", `\S*\b(a|\s*b)c()`, "éac", ECMAScript | Unicode},
		{"scoped Unicode word", `\S*(?i:\b)(K|\s*b)c()`, " Kc", ECMAScript | Unicode},
		{"scoped Unicode nonword", `\S*(?-i:\B)(K|\s*b)c()`, " Kc", ECMAScript | Unicode | IgnoreCase},
	} {
		t.Run(tt.name, func(t *testing.T) {
			re := MustCompile(tt.pattern, tt.options)
			if re.code.FindOptimizations == nil || re.code.FindOptimizations.FindMode != syntax.RequiredLandmarkChain_LeftToRight {
				t.Fatal("fixture must exercise required-landmark filtering")
			}

			matched, err := re.MatchString(tt.input)

			if err != nil || !matched {
				t.Fatalf("%s on %q = %v, %v; want true, nil", tt.pattern, tt.input, matched, err)
			}
		})
	}
}

// The following literal must remain reachable when the loop backtracks across
// a boundary whose scoped word class differs from the loop's own flags.
func TestECMAWordBoundaryAtomicLoop(t *testing.T) {
	for _, tt := range []struct {
		name, pattern, input string
	}{
		{"literal loop", `K+(?-i:\B)K`, "KK"},
		{"class loop", `[k]+(?-i:\B)K`, "KK"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			re := MustCompile(tt.pattern, ECMAScript|Unicode|IgnoreCase)

			matched, err := re.MatchString(tt.input)

			if err != nil || !matched {
				t.Fatalf("%s on %q = %v, %v; want true, nil", tt.pattern, tt.input, matched, err)
			}
		})
	}
}

func TestECMAWordBoundaryRuntimeHelpers(t *testing.T) {
	for _, tt := range []struct {
		name, input string
		options     RegexOptions
		global      bool
		scoped      bool
	}{
		{"global off scoped on", "K", ECMAScript | Unicode, false, true},
		{"global on scoped off", "K", ECMAScript | Unicode | IgnoreCase, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			re := MustCompile("", tt.options)
			r := &Runner{re: re, Runtext: []rune(tt.input), Runtextend: 1}
			if got := r.IsECMABoundary(0); got != tt.global {
				t.Fatalf("global helper = %v, want %v", got, tt.global)
			}

			scoped := tt.options ^ IgnoreCase
			got := r.IsECMABoundaryWithOptions(0, scoped)

			if got != tt.scoped {
				t.Fatalf("scoped helper = %v, want %v", got, tt.scoped)
			}
			if after := r.IsECMABoundary(0); after != tt.global {
				t.Fatalf("scoped helper changed global state: %v, want %v", after, tt.global)
			}
		})
	}
}
