package cli

// Documentation-site deep-links referenced from --help. Keeping them in one
// place means a domain move only touches this file, and the tripwire test can
// assert that every business command links back to the site.
const (
	docsBaseURL            = "https://yewfence.github.io/YewSeal"
	docsTutorial           = docsBaseURL + "/guide/tutorial"
	docsConfiguration      = docsBaseURL + "/guide/configuration"
	docsTargetSelect       = docsBaseURL + "/guide/target-selection"
	docsDecryptResults     = docsBaseURL + "/guide/decryption-results"
	docsPlaintextCleanup   = docsBaseURL + "/guide/plaintext-cleanup"
	docsReadingPrivateKeys = docsConfiguration + "#reading-private-keys"
	docsPrivateKeys        = docsBaseURL + "/guide/private-keys"
	docsSOPS               = docsBaseURL + "/guide/sops"
	docsCICD               = docsBaseURL + "/guide/ci-cd"
	docsVerify             = docsCICD + "#checking-repository-health"
)
