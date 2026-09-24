package sopsx

import "errors"

// ErrNoMatchingIdentity means no supplied identity could unlock the file.
// It does not establish the integrity of the encrypted contents.
var ErrNoMatchingIdentity = errors.New("no matching age identity; encrypted content was not verified")

// ErrMACMismatch means the file's MAC check failed after decryption.
// This indicates the ciphertext may have been tampered with.
var ErrMACMismatch = errors.New("MAC mismatch: file may have been tampered with")
