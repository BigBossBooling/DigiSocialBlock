package nexuserrors

import "errors"

// General errors
var (
	ErrInternal        = errors.New("internal server error")
	ErrInvalidInput    = errors.New("invalid input")
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrRateLimited     = errors.New("rate limited")
	ErrFeatureDisabled = errors.New("feature disabled")
)

// Validation errors for core types
var (
	ErrValidation      = errors.New("validation failed") // Generic validation error

	// NexusContentObjectV1 validation errors
	ErrContentIDMissing         = errors.New("content ID is missing")
	ErrContentIDInvalidFormat   = errors.New("content ID has invalid format")
	ErrAuthorDIDMissing         = errors.New("author DID is missing")
	ErrAuthorDIDInvalidFormat   = errors.New("author DID has invalid format")
	ErrContentTypeUnspecified   = errors.New("content type is unspecified")
	ErrContentBodyMissing       = errors.New("content body is missing")
	ErrContentBodyTooLong       = errors.New("content body is too long")
	ErrTimestampInvalid         = errors.New("timestamp is invalid") // Generic timestamp error
	ErrClientTimestampMissing   = errors.New("client-asserted timestamp is missing")
	ErrNetworkTimestampInvalid  = errors.New("network-observed timestamp is invalid (e.g., in the future)")
	ErrPreviousVersionIDFormat  = errors.New("previous version ID has invalid format")
	ErrLicenseURITooLong        = errors.New("license URI is too long")
	ErrLicenseURIInvalidFormat  = errors.New("license URI has invalid format")
	ErrCustomMetadataKeyTooLong = errors.New("custom metadata key is too long")
	ErrCustomMetadataValueTooLong = errors.New("custom metadata value is too long")
	ErrTooManyTags              = errors.New("too many tags")
	ErrTagTooLong               = errors.New("tag is too long")

	// NexusUserObjectV1 validation errors
	ErrUserDIDMissing             = errors.New("user DID is missing")
	ErrUserDIDInvalidFormat       = errors.New("user DID has invalid format")
	ErrDisplayNameMissing         = errors.New("display name is missing")
	ErrDisplayNameTooLong         = errors.New("display name is too long")
	ErrDisplayNameInvalidChars  = errors.New("display name contains invalid characters")
	ErrBioTooLong                 = errors.New("bio is too long")
	ErrAvatarURITooLong           = errors.New("avatar URI is too long")
	ErrAvatarURIInvalidFormat     = errors.New("avatar URI has invalid format")
	ErrBannerURITooLong           = errors.New("banner URI is too long")
	ErrBannerURIInvalidFormat     = errors.New("banner URI has invalid format")
	ErrProfileLinkKeyTooLong      = errors.New("profile link key is too long")
	ErrProfileLinkValueTooLong    = errors.New("profile link value is too long")
	ErrProfileLinkInvalidFormat   = errors.New("profile link has invalid format")
	ErrTooManyProfileLinks        = errors.New("too many profile links")

	// NexusInteractionRecordV1 validation errors
	ErrInteractionIDMissing         = errors.New("interaction ID is missing")
	ErrInteractionIDInvalidFormat   = errors.New("interaction ID has invalid format")
	// UserDIDMissing is covered by ErrAuthorDIDMissing or ErrUserDIDMissing
	// UserDIDInvalidFormat is covered by ErrAuthorDIDInvalidFormat or ErrUserDIDInvalidFormat
	ErrTargetContentIDMissing       = errors.New("target content ID is missing")
	ErrTargetContentIDInvalidFormat = errors.New("target content ID has invalid format")
	ErrInteractionTypeUnspecified = errors.New("interaction type is unspecified")
	ErrInteractionDataTooLong     = errors.New("interaction data is too long")
	// ClientTimestampMissing is covered by ErrClientTimestampMissing
	// NetworkTimestampInvalid is covered by ErrNetworkTimestampInvalid
	ErrReferenceInteractionIDFormat = errors.New("reference interaction ID has invalid format")

	// WitnessProofV1 validation errors
	ErrProofIDMissing                 = errors.New("proof ID is missing")
	ErrProofIDInvalidFormat           = errors.New("proof ID has invalid format")
	ErrWitnessDIDMissing              = errors.New("witness DID is missing")
	ErrWitnessDIDInvalidFormat        = errors.New("witness DID has invalid format")
	ErrObservedContentIDMissing       = errors.New("observed content ID is missing")
	ErrObservedContentIDInvalidFormat = errors.New("observed content ID has invalid format")
	ErrObservedContentHashMissing     = errors.New("observed content hash is missing")
	ErrObservedContentHashInvalid     = errors.New("observed content hash is invalid")
	ErrWitnessTimestampMissing        = errors.New("witness-observed timestamp is missing")
	ErrWitnessTimestampInFuture       = errors.New("witness-observed timestamp is in the future")
	ErrSignatureMissing               = errors.New("signature is missing")
	ErrSignatureInvalid               = errors.New("signature is invalid")
	ErrProofStatusUnspecified         = errors.New("proof status is unspecified")
	ErrVerifierDIDInvalidFormat       = errors.New("verifier DID has invalid format")
	ErrVerifiedTimestampInvalid       = errors.New("verified timestamp is invalid (e.g., before witness observation)")
)

// TODO: Add more specific errors as needed for other modules (DDS, PoW, etc.)

// Errorf creates a new error with a formatted message.
// This is a simple wrapper around fmt.Errorf, but could be extended
// to include more context or structured error data if needed.
// For now, we'll rely on distinct error variables.
// func Errorf(format string, args ...interface{}) error {
// 	return fmt.Errorf(format, args...)
// }

// Is checks if an error is of a specific type.
// Useful for checking against the global error variables.
// Example: if nexuserrors.Is(err, nexuserrors.ErrInvalidInput) { ... }
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Wrap could be used to add context to an error, but for now, standard library's
// fmt.Errorf with %w verb is preferred for simple wrapping.
// If more complex error wrapping or stacking is needed, this can be expanded.
// func Wrap(err error, message string) error {
// 	if err == nil {
// 		return nil
// 	}
// 	return fmt.Errorf("%s: %w", message, err)
// }
