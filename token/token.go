package token

// Token represents a lexical element.
type Token int

const(
	// Special tokens
	ILLEGAL Token = iota
	EOF
	COMMENT // G1: '#?' multiline or G2: // single line

	// Identifiers and basic type literals
	IDENT // e.g. title
	TEXT // in G1 just a run of text or "abc"

)
