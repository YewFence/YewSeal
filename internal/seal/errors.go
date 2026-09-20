package seal

import "errors"

// ErrNoIdentity means decryption could not start because no identity was supplied.
var ErrNoIdentity = errors.New("no age identity is available; encrypted content was not verified")
