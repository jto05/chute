module github.com/jto05/chute

go 1.24.1

require github.com/am29/ferdinand v0.0.0

// Local development: point to the sibling repo.
// Remove this line before deploying; replace with a tagged version.
replace github.com/am29/ferdinand => ../ferdinand
