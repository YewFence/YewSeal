package sopsx

import "errors"

// ErrNoMatchingIdentity means no supplied identity could unlock the file.
// It does not establish the integrity of the encrypted contents.
var ErrNoMatchingIdentity = errors.New("no matching age identity; encrypted content was not verified")
