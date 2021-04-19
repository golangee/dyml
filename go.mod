module github.com/golangee/tadl

go 1.16


require (
	github.com/alecthomas/participle/v2 v2.0.0-alpha4
	golang.org/x/mod v0.4.2
)

// TODO merge bugfix upstream
replace github.com/alecthomas/participle/v2  => ../../torbenschinke/participle
