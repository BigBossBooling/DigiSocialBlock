package types

import (
	"strings"
	"testing"
	"time"

	"github.com/DigiSocialBlock/nexus-protocol/internal/nexuserrors"
	"github.com/stretchr/testify/assert"
)

// --- Helper Functions ---
func validNexusContentObjectV1() *NexusContentObjectV1 {
	return &NexusContentObjectV1{
		ContentId:         "content" + strings.Repeat("0", 19), // 20 chars
		AuthorDid:         "did:example:author123",
		ContentType:       ContentType_CONTENT_TYPE_POST,
		ContentBody:       "Valid content body.",
		Tags:              []string{"valid", "tag"},
		CreatedAtClient:   time.Now().UnixNano(),
		CreatedAtNetwork:  time.Now().UnixNano() + 1000, // Slightly after client
		PreviousVersionId: "",
		LicenseUri:        "https://example.com/license",
		CustomMetadata:    map[string]string{"key": "value"},
	}
}

func validNexusUserObjectV1() *NexusUserObjectV1 {
	return &NexusUserObjectV1{
		UserDid:      "did:example:user456",
		DisplayName:  "Valid User",
		Bio:          "Valid bio.",
		AvatarUri:    "https://example.com/avatar.png",
		BannerUri:    "https://example.com/banner.png",
		CreatedAt:    time.Now().UnixNano(),
		UpdatedAt:    time.Now().UnixNano() + 1000,
		ProfileLinks: map[string]string{"website": "https://example.com"},
		IsVerified:   false,
	}
}

func validNexusInteractionRecordV1() *NexusInteractionRecordV1 {
	return &NexusInteractionRecordV1{
		InteractionId:          "interaction" + strings.Repeat("0", 11), // 20 chars
		UserDid:                "did:example:user789",
		TargetContentId:        "content" + strings.Repeat("1", 19), // 20 chars
		InteractionType:        InteractionType_INTERACTION_TYPE_LIKE,
		InteractionData:        "Optional data",
		CreatedAtClient:        time.Now().UnixNano(),
		CreatedAtNetwork:       time.Now().UnixNano() + 1000,
		ReferenceInteractionId: "",
	}
}

func validWitnessProofV1() *WitnessProofV1 {
	now := time.Now().UnixNano()
	return &WitnessProofV1{
		ProofId:             "proof" + strings.Repeat("0", 15), // 20 chars
		WitnessDid:          "did:example:witness101",
		ObservedContentId:   "content" + strings.Repeat("2", 19), // 20 chars
		ObservedContentHash: "sha256-validhashplaceholder",
		ObservedAtWitness:   now,
		Signature:           "valid_signature_placeholder",
		Status:              WitnessProofStatus_WITNESS_PROOF_STATUS_PENDING,
		VerifiedAt:          0, // Not verified yet
		VerifierDid:         "",  // No verifier yet
	}
}


// --- Test Cases ---

func TestNexusContentObjectV1_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(obj *NexusContentObjectV1)
		expectedErr error
	}{
		{"valid object", func(obj *NexusContentObjectV1) {}, nil},
		{"nil object", func(obj *NexusContentObjectV1) { /* Special case handled by caller */ }, nexuserrors.ErrInvalidInput},
		{"missing content_id", func(obj *NexusContentObjectV1) { obj.ContentId = "" }, nexuserrors.ErrContentIDMissing},
		{"invalid content_id format", func(obj *NexusContentObjectV1) { obj.ContentId = "short" }, nexuserrors.ErrContentIDInvalidFormat},
		{"missing author_did", func(obj *NexusContentObjectV1) { obj.AuthorDid = "" }, nexuserrors.ErrAuthorDIDMissing},
		{"invalid author_did format", func(obj *NexusContentObjectV1) { obj.AuthorDid = "invalid:did" }, nexuserrors.ErrAuthorDIDInvalidFormat},
		{"unspecified content_type", func(obj *NexusContentObjectV1) { obj.ContentType = ContentType_CONTENT_TYPE_UNSPECIFIED }, nexuserrors.ErrContentTypeUnspecified},
		{"content_body too long", func(obj *NexusContentObjectV1) { obj.ContentBody = strings.Repeat("a", maxContentBodyLength+1) }, nexuserrors.ErrContentBodyTooLong},
		{"too many tags", func(obj *NexusContentObjectV1) { obj.Tags = make([]string, maxTagsCount+1) }, nexuserrors.ErrTooManyTags},
		{"tag too long", func(obj *NexusContentObjectV1) { obj.Tags = []string{strings.Repeat("t", maxTagLength+1)} }, nexuserrors.ErrTagTooLong},
		{"empty tag", func(obj *NexusContentObjectV1) { obj.Tags = []string{" "} }, nexuserrors.ErrTagTooLong}, // Assuming ErrTagTooLong for now
		{"missing created_at_client", func(obj *NexusContentObjectV1) { obj.CreatedAtClient = 0 }, nexuserrors.ErrClientTimestampMissing},
		{"invalid created_at_network (negative)", func(obj *NexusContentObjectV1) { obj.CreatedAtNetwork = -1 }, nexuserrors.ErrNetworkTimestampInvalid},
		{"invalid previous_version_id format", func(obj *NexusContentObjectV1) { obj.PreviousVersionId = "short" }, nexuserrors.ErrPreviousVersionIDFormat},
		{"license_uri too long", func(obj *NexusContentObjectV1) { obj.LicenseUri = "https://" + strings.Repeat("a", maxLicenseURILength) + ".com" }, nexuserrors.ErrLicenseURITooLong},
		{"license_uri invalid format", func(obj *NexusContentObjectV1) { obj.LicenseUri = "invaliduri" }, nexuserrors.ErrLicenseURIInvalidFormat},
		{"custom_metadata key too long", func(obj *NexusContentObjectV1) { obj.CustomMetadata = map[string]string{strings.Repeat("k", maxCustomMetadataKeyLen+1): "v"} }, nexuserrors.ErrCustomMetadataKeyTooLong},
		{"custom_metadata value too long", func(obj *NexusContentObjectV1) { obj.CustomMetadata = map[string]string{"k": strings.Repeat("v", maxCustomMetadataValueLen+1)} }, nexuserrors.ErrCustomMetadataValueTooLong},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var obj *NexusContentObjectV1
			if tt.name == "nil object" {
				// obj remains nil
			} else {
				obj = validNexusContentObjectV1()
				tt.modifier(obj)
			}
			err := obj.Validate()
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}


func TestNexusUserObjectV1_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		modifier    func(obj *NexusUserObjectV1)
		expectedErr error
	}{
		{"valid object", func(obj *NexusUserObjectV1) {}, nil},
		{"nil object", func(obj *NexusUserObjectV1) { /* Special case */ }, nexuserrors.ErrInvalidInput},
		{"missing user_did", func(obj *NexusUserObjectV1) { obj.UserDid = "" }, nexuserrors.ErrUserDIDMissing},
		{"invalid user_did format", func(obj *NexusUserObjectV1) { obj.UserDid = "invalid" }, nexuserrors.ErrUserDIDInvalidFormat},
		{"missing display_name", func(obj *NexusUserObjectV1) { obj.DisplayName = " " }, nexuserrors.ErrDisplayNameMissing},
		{"display_name too long", func(obj *NexusUserObjectV1) { obj.DisplayName = strings.Repeat("n", maxDisplayNameLength+1) }, nexuserrors.ErrDisplayNameTooLong},
		{"bio too long", func(obj *NexusUserObjectV1) { obj.Bio = strings.Repeat("b", maxBioLength+1) }, nexuserrors.ErrBioTooLong},
		{"avatar_uri too long", func(obj *NexusUserObjectV1) { obj.AvatarUri = "https://" + strings.Repeat("a", maxAvatarURILength) + ".com" }, nexuserrors.ErrAvatarURITooLong},
		{"avatar_uri invalid format", func(obj *NexusUserObjectV1) { obj.AvatarUri = "invalid" }, nexuserrors.ErrAvatarURIInvalidFormat},
		{"banner_uri too long", func(obj *NexusUserObjectV1) { obj.BannerUri = "https://" + strings.Repeat("b", maxBannerURILength) + ".com" }, nexuserrors.ErrBannerURITooLong},
		{"banner_uri invalid format", func(obj *NexusUserObjectV1) { obj.BannerUri = "invalid" }, nexuserrors.ErrBannerURIInvalidFormat},
		{"invalid created_at (zero, but not for new record)", func(obj *NexusUserObjectV1) { obj.CreatedAt = 0 }, nexuserrors.ErrTimestampInvalid},
		{"invalid updated_at (negative)", func(obj *NexusUserObjectV1) { obj.UpdatedAt = -1 }, nexuserrors.ErrTimestampInvalid},
		{"updated_at before created_at", func(obj *NexusUserObjectV1) { obj.CreatedAt = time.Now().UnixNano(); obj.UpdatedAt = obj.CreatedAt - 1000 }, nexuserrors.ErrTimestampInvalid},
		{"too many profile_links", func(obj *NexusUserObjectV1) { obj.ProfileLinks = make(map[string]string, maxProfileLinksCount+1); for i:=0; i<maxProfileLinksCount+1; i++ { obj.ProfileLinks[string(rune(i))] = "https://example.com"} }, nexuserrors.ErrTooManyProfileLinks},
		{"profile_links key too long", func(obj *NexusUserObjectV1) { obj.ProfileLinks = map[string]string{strings.Repeat("k", maxProfileLinkKeyLength+1): "https://example.com"} }, nexuserrors.ErrProfileLinkKeyTooLong},
		{"profile_links value too long", func(obj *NexusUserObjectV1) { obj.ProfileLinks = map[string]string{"key": "https://" + strings.Repeat("v", maxProfileLinkValueLength) + ".com"} }, nexuserrors.ErrProfileLinkValueTooLong},
		{"profile_links value invalid format", func(obj *NexusUserObjectV1) { obj.ProfileLinks = map[string]string{"key": "invalid"} }, nexuserrors.ErrProfileLinkInvalidFormat},
		{"profile_links empty key", func(obj *NexusUserObjectV1) { obj.ProfileLinks = map[string]string{" ": "https://example.com"} }, nexuserrors.ErrProfileLinkKeyTooLong}, // Assuming same error
		{"profile_links empty value", func(obj *NexusUserObjectV1) { obj.ProfileLinks = map[string]string{"key": " "} }, nexuserrors.ErrProfileLinkValueTooLong}, // Assuming same error
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var obj *NexusUserObjectV1
			if tt.name == "nil object" {
			} else {
				obj = validNexusUserObjectV1()
				tt.modifier(obj)
			}
			err := obj.Validate()
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}

func TestNexusInteractionRecordV1_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		modifier    func(obj *NexusInteractionRecordV1)
		expectedErr error
	}{
		{"valid object", func(obj *NexusInteractionRecordV1) {}, nil},
		{"nil object", func(obj *NexusInteractionRecordV1) { /* Special case */ }, nexuserrors.ErrInvalidInput},
		{"missing interaction_id", func(obj *NexusInteractionRecordV1) { obj.InteractionId = "" }, nexuserrors.ErrInteractionIDMissing},
		{"invalid interaction_id format", func(obj *NexusInteractionRecordV1) { obj.InteractionId = "short" }, nexuserrors.ErrInteractionIDInvalidFormat},
		{"missing user_did", func(obj *NexusInteractionRecordV1) { obj.UserDid = "" }, nexuserrors.ErrUserDIDMissing},
		{"invalid user_did format", func(obj *NexusInteractionRecordV1) { obj.UserDid = "invalid" }, nexuserrors.ErrUserDIDInvalidFormat},
		{"missing target_content_id", func(obj *NexusInteractionRecordV1) { obj.TargetContentId = "" }, nexuserrors.ErrTargetContentIDMissing},
		{"invalid target_content_id format", func(obj *NexusInteractionRecordV1) { obj.TargetContentId = "short" }, nexuserrors.ErrTargetContentIDInvalidFormat},
		{"unspecified interaction_type", func(obj *NexusInteractionRecordV1) { obj.InteractionType = InteractionType_INTERACTION_TYPE_UNSPECIFIED }, nexuserrors.ErrInteractionTypeUnspecified},
		{"interaction_data too long", func(obj *NexusInteractionRecordV1) { obj.InteractionData = strings.Repeat("d", maxInteractionDataLength+1) }, nexuserrors.ErrInteractionDataTooLong},
		{"missing created_at_client", func(obj *NexusInteractionRecordV1) { obj.CreatedAtClient = 0 }, nexuserrors.ErrClientTimestampMissing},
		{"invalid created_at_network (negative)", func(obj *NexusInteractionRecordV1) { obj.CreatedAtNetwork = -1 }, nexuserrors.ErrNetworkTimestampInvalid},
		{"invalid reference_interaction_id format", func(obj *NexusInteractionRecordV1) { obj.ReferenceInteractionId = "short" }, nexuserrors.ErrReferenceInteractionIDFormat},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var obj *NexusInteractionRecordV1
			if tt.name == "nil object" {
			} else {
				obj = validNexusInteractionRecordV1()
				tt.modifier(obj)
			}
			err := obj.Validate()
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}

func TestWitnessProofV1_Validate(t *testing.T) {
	t.Parallel()
	now := time.Now()
	validObservedAt := now.UnixNano()

	tests := []struct {
		name        string
		modifier    func(obj *WitnessProofV1)
		expectedErr error
	}{
		{"valid object", func(obj *WitnessProofV1) {}, nil},
		{"nil object", func(obj *WitnessProofV1) { /* Special case */ }, nexuserrors.ErrInvalidInput},
		{"missing proof_id", func(obj *WitnessProofV1) { obj.ProofId = "" }, nexuserrors.ErrProofIDMissing},
		{"invalid proof_id format", func(obj *WitnessProofV1) { obj.ProofId = "short" }, nexuserrors.ErrProofIDInvalidFormat},
		{"missing witness_did", func(obj *WitnessProofV1) { obj.WitnessDid = "" }, nexuserrors.ErrWitnessDIDMissing},
		{"invalid witness_did format", func(obj *WitnessProofV1) { obj.WitnessDid = "invalid" }, nexuserrors.ErrWitnessDIDInvalidFormat},
		{"missing observed_content_id", func(obj *WitnessProofV1) { obj.ObservedContentId = "" }, nexuserrors.ErrObservedContentIDMissing},
		{"invalid observed_content_id format", func(obj *WitnessProofV1) { obj.ObservedContentId = "short" }, nexuserrors.ErrObservedContentIDInvalidFormat},
		{"missing observed_content_hash", func(obj *WitnessProofV1) { obj.ObservedContentHash = "" }, nexuserrors.ErrObservedContentHashMissing},
		// {"invalid observed_content_hash format", func(obj *WitnessProofV1) { obj.ObservedContentHash = "invalidhash" }, nexuserrors.ErrObservedContentHashInvalid}, // Requires hashRegex
		{"missing observed_at_witness", func(obj *WitnessProofV1) { obj.ObservedAtWitness = 0 }, nexuserrors.ErrWitnessTimestampMissing},
		{"observed_at_witness in future", func(obj *WitnessProofV1) { obj.ObservedAtWitness = now.Add(10 * time.Minute).UnixNano() }, nexuserrors.ErrWitnessTimestampInFuture},
		{"missing signature", func(obj *WitnessProofV1) { obj.Signature = "" }, nexuserrors.ErrSignatureMissing},
		{"unspecified status", func(obj *WitnessProofV1) { obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_UNSPECIFIED }, nexuserrors.ErrProofStatusUnspecified},
		{"invalid verifier_did format (when present)", func(obj *WitnessProofV1) { obj.VerifierDid = "invalid"; obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED; obj.VerifiedAt = validObservedAt + 1000 }, nexuserrors.ErrVerifierDIDInvalidFormat},
		{"invalid verified_at (negative)", func(obj *WitnessProofV1) { obj.VerifiedAt = -1; obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED; obj.VerifierDid = "did:example:v1" }, nexuserrors.ErrVerifiedTimestampInvalid},
		{"verified_at missing when status is VERIFIED", func(obj *WitnessProofV1) { obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED; obj.VerifiedAt = 0; obj.VerifierDid = "did:example:v1" }, nexuserrors.ErrVerifiedTimestampInvalid},
		{"verified_at before observed_at_witness", func(obj *WitnessProofV1) { obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED; obj.VerifiedAt = validObservedAt - 1000; obj.VerifierDid = "did:example:v1" }, nexuserrors.ErrVerifiedTimestampInvalid},
		{"verifier_did missing when status is VERIFIED", func(obj *WitnessProofV1) { obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED; obj.VerifiedAt = validObservedAt + 1000; obj.VerifierDid = " " }, nexuserrors.ErrVerifierDIDInvalidFormat},
		{"verified_at set when status is PENDING", func(obj *WitnessProofV1) { obj.Status = WitnessProofStatus_WITNESS_PROOF_STATUS_PENDING; obj.VerifiedAt = validObservedAt + 1000 }, nexuserrors.ErrVerifiedTimestampInvalid},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var obj *WitnessProofV1
			if tt.name == "nil object" {
			} else {
				obj = validWitnessProofV1()
				obj.ObservedAtWitness = validObservedAt // Standardize for tests that modify it
				tt.modifier(obj)
			}
			err := obj.Validate()
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr, "Error mismatch for test: %s. Expected %v, got %v", tt.name, tt.expectedErr, err)
			}
		})
	}
}
