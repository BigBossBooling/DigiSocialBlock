package types

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/DigiSocialBlock/nexus-protocol/internal/nexuserrors"
)

const (
	maxContentBodyLength      = 100000 // Example: 100KB
	maxTagsCount              = 50
	maxTagLength              = 50
	maxLicenseURILength       = 2048
	maxCustomMetadataKeyLen   = 128
	maxCustomMetadataValueLen = 4096
	maxDisplayNameLength      = 100
	maxBioLength              = 500
	maxAvatarURILength        = 2048
	maxBannerURILength        = 2048
	maxProfileLinksCount      = 20
	maxProfileLinkKeyLength   = 50
	maxProfileLinkValueLength = 2048
	maxInteractionDataLength  = 10000 // Example: 10KB
)

// Basic DID format check (very simplistic, replace with actual DID validation logic)
var didRegex = regexp.MustCompile(`^did:[a-z0-9]+:[a-zA-Z0-9.-_]+$`)

// Basic ID format check (e.g. CUID, UUID, or hash representation) - adjust as needed
var idRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{20,256}$`) // Example: allowing for CUIDs, hashes etc.

// URI format check (simplistic)
var uriRegex = regexp.MustCompile(`^(https?|ipfs|ipns)://[^\s/$.?#].[^\s]*$`)


// --- NexusContentObjectV1 ---

// Validate checks the validity of the NexusContentObjectV1.
func (x *NexusContentObjectV1) Validate() error {
	if x == nil {
		return nexuserrors.ErrInvalidInput // Or a more specific error like ErrNilObject
	}

	if strings.TrimSpace(x.ContentId) == "" {
		return nexuserrors.ErrContentIDMissing
	}
	if !idRegex.MatchString(x.ContentId) {
		return nexuserrors.ErrContentIDInvalidFormat
	}

	if strings.TrimSpace(x.AuthorDid) == "" {
		return nexuserrors.ErrAuthorDIDMissing
	}
	if !didRegex.MatchString(x.AuthorDid) {
		return nexuserrors.ErrAuthorDIDInvalidFormat
	}

	if x.ContentType == ContentType_CONTENT_TYPE_UNSPECIFIED {
		return nexuserrors.ErrContentTypeUnspecified
	}

	// ContentBody might be optional for certain content types (e.g. a simple share with no comment)
	// Add specific checks if required for certain types.
	if utf8.RuneCountInString(x.ContentBody) > maxContentBodyLength {
		return nexuserrors.ErrContentBodyTooLong
	}

	if len(x.Tags) > maxTagsCount {
		return nexuserrors.ErrTooManyTags
	}
	for _, tag := range x.Tags {
		if strings.TrimSpace(tag) == "" {
			// Or allow empty tags if that's valid for the system
			return nexuserrors.ErrTagTooLong // Re-using, or create ErrTagEmpty
		}
		if utf8.RuneCountInString(tag) > maxTagLength {
			return nexuserrors.ErrTagTooLong
		}
	}

	if x.CreatedAtClient <= 0 {
		return nexuserrors.ErrClientTimestampMissing
	}
	// Could add a check against current time, allowing for some clock skew
	// if time.Unix(0, x.CreatedAtClient).After(time.Now().Add(5 * time.Minute)) {
	// 	return nexuserrors.ErrTimestampInvalid // Or a more specific one like ErrClientTimestampInFuture
	// }

	// CreatedAtNetwork is set by the network, so client validation might only check if it's present when expected
	// For now, we assume it can be 0 if not yet processed by the network.
	if x.CreatedAtNetwork < 0 { // Should not be negative
		return nexuserrors.ErrNetworkTimestampInvalid
	}
	if x.CreatedAtNetwork > 0 && x.CreatedAtNetwork < x.CreatedAtClient {
		// This could be a sign of an issue, but clock skew can make it tricky.
		// Depending on strictness, this might be an error.
		// return nexuserrors.ErrNetworkTimestampInvalid
	}


	if x.PreviousVersionId != "" && !idRegex.MatchString(x.PreviousVersionId) {
		return nexuserrors.ErrPreviousVersionIDFormat
	}

	if x.LicenseUri != "" {
		if utf8.RuneCountInString(x.LicenseUri) > maxLicenseURILength {
			return nexuserrors.ErrLicenseURITooLong
		}
		if !uriRegex.MatchString(x.LicenseUri) {
			return nexuserrors.ErrLicenseURIInvalidFormat
		}
	}

	if x.CustomMetadata != nil {
		for k, v := range x.CustomMetadata {
			if utf8.RuneCountInString(k) > maxCustomMetadataKeyLen {
				return nexuserrors.ErrCustomMetadataKeyTooLong
			}
			if utf8.RuneCountInString(v) > maxCustomMetadataValueLen {
				return nexuserrors.ErrCustomMetadataValueTooLong
			}
		}
	}

	return nil
}

// --- NexusUserObjectV1 ---

// Validate checks the validity of the NexusUserObjectV1.
func (x *NexusUserObjectV1) Validate() error {
	if x == nil {
		return nexuserrors.ErrInvalidInput
	}

	if strings.TrimSpace(x.UserDid) == "" {
		return nexuserrors.ErrUserDIDMissing
	}
	if !didRegex.MatchString(x.UserDid) {
		return nexuserrors.ErrUserDIDInvalidFormat
	}

	if strings.TrimSpace(x.DisplayName) == "" {
		return nexuserrors.ErrDisplayNameMissing
	}
	if utf8.RuneCountInString(x.DisplayName) > maxDisplayNameLength {
		return nexuserrors.ErrDisplayNameTooLong
	}
	// Add character validation for display name if needed

	if utf8.RuneCountInString(x.Bio) > maxBioLength {
		return nexuserrors.ErrBioTooLong
	}

	if x.AvatarUri != "" {
		if utf8.RuneCountInString(x.AvatarUri) > maxAvatarURILength {
			return nexuserrors.ErrAvatarURITooLong
		}
		if !uriRegex.MatchString(x.AvatarUri) {
			return nexuserrors.ErrAvatarURIInvalidFormat
		}
	}

	if x.BannerUri != "" {
		if utf8.RuneCountInString(x.BannerUri) > maxBannerURILength {
			return nexuserrors.ErrBannerURITooLong
		}
		if !uriRegex.MatchString(x.BannerUri) {
			return nexuserrors.ErrBannerURIInvalidFormat
		}
	}

	// For an existing record, CreatedAt should generally not be 0.
	// The test "invalid_created_at_(zero,_but_not_for_new_record)" implies 0 is invalid.
	// If 0 is a valid state for a "new" unpersisted object, that logic is outside Validate().
	if x.CreatedAt == 0 {
		return nexuserrors.ErrTimestampInvalid
	}
  if x.CreatedAt < 0 { // Negative timestamps are definitively invalid
    return nexuserrors.ErrTimestampInvalid
  }

	if x.UpdatedAt < 0 { // UpdatedAt can be 0 if never updated
		return nexuserrors.ErrTimestampInvalid
	}
	if x.UpdatedAt > 0 && x.UpdatedAt < x.CreatedAt {
		return nexuserrors.ErrTimestampInvalid // UpdatedAt cannot be before CreatedAt
	}


	if len(x.ProfileLinks) > maxProfileLinksCount {
		return nexuserrors.ErrTooManyProfileLinks
	}
	for k, v := range x.ProfileLinks {
		if strings.TrimSpace(k) == "" {
			return nexuserrors.ErrProfileLinkKeyTooLong // Or ErrProfileLinkKeyEmpty
		}
		if utf8.RuneCountInString(k) > maxProfileLinkKeyLength {
			return nexuserrors.ErrProfileLinkKeyTooLong
		}
		if strings.TrimSpace(v) == "" {
			return nexuserrors.ErrProfileLinkValueTooLong // Or ErrProfileLinkValueEmpty
		}
		if utf8.RuneCountInString(v) > maxProfileLinkValueLength {
			return nexuserrors.ErrProfileLinkValueTooLong
		}
		if !uriRegex.MatchString(v) { // Assuming profile links are URIs
			return nexuserrors.ErrProfileLinkInvalidFormat
		}
	}
	// IsVerified is a boolean, no specific validation other than type.

	return nil
}

// --- NexusInteractionRecordV1 ---

// Validate checks the validity of the NexusInteractionRecordV1.
func (x *NexusInteractionRecordV1) Validate() error {
	if x == nil {
		return nexuserrors.ErrInvalidInput
	}

	if strings.TrimSpace(x.InteractionId) == "" {
		return nexuserrors.ErrInteractionIDMissing
	}
	if !idRegex.MatchString(x.InteractionId) {
		return nexuserrors.ErrInteractionIDInvalidFormat
	}

	if strings.TrimSpace(x.UserDid) == "" {
		return nexuserrors.ErrUserDIDMissing // Re-using UserDID error
	}
	if !didRegex.MatchString(x.UserDid) {
		return nexuserrors.ErrUserDIDInvalidFormat // Re-using
	}

	if strings.TrimSpace(x.TargetContentId) == "" {
		return nexuserrors.ErrTargetContentIDMissing
	}
	if !idRegex.MatchString(x.TargetContentId) {
		return nexuserrors.ErrTargetContentIDInvalidFormat
	}

	if x.InteractionType == InteractionType_INTERACTION_TYPE_UNSPECIFIED {
		return nexuserrors.ErrInteractionTypeUnspecified
	}

	if utf8.RuneCountInString(x.InteractionData) > maxInteractionDataLength {
		return nexuserrors.ErrInteractionDataTooLong
	}
	// Specific validation for InteractionData based on InteractionType might be needed.
	// e.g., if type is COMMENT, InteractionData should not be empty.

	if x.CreatedAtClient <= 0 {
		return nexuserrors.ErrClientTimestampMissing
	}

	if x.CreatedAtNetwork < 0 {
		return nexuserrors.ErrNetworkTimestampInvalid
	}
	// Similar timestamp logic as NexusContentObjectV1

	if x.ReferenceInteractionId != "" && !idRegex.MatchString(x.ReferenceInteractionId) {
		return nexuserrors.ErrReferenceInteractionIDFormat
	}

	return nil
}

// --- WitnessProofV1 ---

// Validate checks the validity of the WitnessProofV1.
func (x *WitnessProofV1) Validate() error {
	if x == nil {
		return nexuserrors.ErrInvalidInput
	}

	if strings.TrimSpace(x.ProofId) == "" {
		return nexuserrors.ErrProofIDMissing
	}
	if !idRegex.MatchString(x.ProofId) {
		return nexuserrors.ErrProofIDInvalidFormat
	}

	if strings.TrimSpace(x.WitnessDid) == "" {
		return nexuserrors.ErrWitnessDIDMissing
	}
	if !didRegex.MatchString(x.WitnessDid) {
		return nexuserrors.ErrWitnessDIDInvalidFormat
	}

	if strings.TrimSpace(x.ObservedContentId) == "" {
		return nexuserrors.ErrObservedContentIDMissing
	}
	if !idRegex.MatchString(x.ObservedContentId) {
		return nexuserrors.ErrObservedContentIDInvalidFormat
	}

	if strings.TrimSpace(x.ObservedContentHash) == "" {
		return nexuserrors.ErrObservedContentHashMissing
	}
	// Add regex for hash format if available, e.g., SHA256 hex
	// if !hashRegex.MatchString(x.ObservedContentHash) {
	// 	return nexuserrors.ErrObservedContentHashInvalid
	// }

	if x.ObservedAtWitness <= 0 {
		return nexuserrors.ErrWitnessTimestampMissing
	}
	if time.Unix(0, x.ObservedAtWitness).After(time.Now().Add(5 * time.Minute)) { // Allow small clock skew
		return nexuserrors.ErrWitnessTimestampInFuture
	}

	if strings.TrimSpace(x.Signature) == "" {
		return nexuserrors.ErrSignatureMissing
	}
	// Signature validation itself is a cryptographic operation, not done here.
	// This just checks for presence and basic format if applicable.

	if x.Status == WitnessProofStatus_WITNESS_PROOF_STATUS_UNSPECIFIED {
		return nexuserrors.ErrProofStatusUnspecified
	}

	if x.VerifierDid != "" && !didRegex.MatchString(x.VerifierDid) {
		return nexuserrors.ErrVerifierDIDInvalidFormat
	}

	if x.VerifiedAt < 0 {
		return nexuserrors.ErrVerifiedTimestampInvalid
	}
	if x.Status == WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED || x.Status == WitnessProofStatus_WITNESS_PROOF_STATUS_REJECTED {
		if x.VerifiedAt == 0 {
			// If it's marked verified/rejected, timestamp should be set
			return nexuserrors.ErrVerifiedTimestampInvalid
		}
		if x.VerifiedAt < x.ObservedAtWitness {
			return nexuserrors.ErrVerifiedTimestampInvalid // Cannot be verified before observed
		}
		if strings.TrimSpace(x.VerifierDid) == "" {
			// If verified/rejected, verifier DID should be set
			return nexuserrors.ErrVerifierDIDInvalidFormat // Or a more specific "VerifierDIDMissingForTerminalStatus"
		}
	}
	if (x.Status == WitnessProofStatus_WITNESS_PROOF_STATUS_PENDING || x.Status == WitnessProofStatus_WITNESS_PROOF_STATUS_UNSPECIFIED) && x.VerifiedAt != 0 {
		// If pending/unspecified, verified_at should not be set
		return nexuserrors.ErrVerifiedTimestampInvalid
	}


	return nil
}
