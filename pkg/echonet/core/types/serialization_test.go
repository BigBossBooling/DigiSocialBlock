package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNexusContentObjectV1_Serialization(t *testing.T) {
	t.Parallel()

	original := &NexusContentObjectV1{
		ContentId:         "content123",
		AuthorDid:         "did:example:author456",
		ContentType:       ContentType_CONTENT_TYPE_POST,
		ContentBody:       "This is a test post.",
		Tags:              []string{"test", "golang", "protobuf"},
		CreatedAtClient:   1678886400000000000, // Example timestamp
		CreatedAtNetwork:  1678886401000000000,
		PreviousVersionId: "content000",
		LicenseUri:        "https://creativecommons.org/licenses/by/4.0/",
		CustomMetadata:    map[string]string{"source": "test-suite", "version": "1.0"},
	}

	t.Run("ToJSON_FromJSON_RoundTrip", func(t *testing.T) {
		t.Parallel()
		jsonString, err := original.ToJSON()
		require.NoError(t, err, "ToJSON() should not produce an error")
		require.NotEmpty(t, jsonString, "JSON string should not be empty")

		// Check if proto names are used (e.g., "content_id" instead of "contentId")
		assert.Contains(t, jsonString, `"content_id":`)
		assert.Contains(t, jsonString, `"author_did":`)
		assert.Contains(t, jsonString, `"content_type":"CONTENT_TYPE_POST"`) // Enum as string

		restored := &NexusContentObjectV1{}
		err = restored.FromJSON(jsonString)
		require.NoError(t, err, "FromJSON() should not produce an error")

		// Use proto.Equal for comparing protobuf messages
		assert.True(t, proto.Equal(original, restored), "Restored object should be equal to the original")
	})

	t.Run("FromJSON_InvalidJSON", func(t *testing.T) {
		t.Parallel()
		invalidJSON := `{"content_id": "123", "contentType": "INVALID_TYPE}` // Malformed JSON
		restored := &NexusContentObjectV1{}
		err := restored.FromJSON(invalidJSON)
		assert.Error(t, err, "FromJSON() with invalid JSON should produce an error")
	})

	t.Run("ToJSON_EmptyObject", func(t *testing.T) {
		t.Parallel()
		emptyObj := &NexusContentObjectV1{}
		jsonString, err := emptyObj.ToJSON()
		require.NoError(t, err)
		// By default, protojson omits unpopulated fields.
		// If EmitUnpopulated were true, it would be "{}".
		// With UseProtoNames and EmitUnpopulated: false, it's "{}" because all are zero/empty.
		// If any field had a default value that wasn't the zero value, it might appear.
		// For our current settings (EmitUnpopulated: false), it should be "{}"
		assert.Equal(t, "{}", jsonString, "JSON string of empty object should be '{}'")


		restored := &NexusContentObjectV1{}
		err = restored.FromJSON(jsonString)
		require.NoError(t, err)
		assert.True(t, proto.Equal(emptyObj, restored))
	})

	t.Run("FromJSON_ExtraFields", func(t *testing.T) {
		t.Parallel()
		jsonWithExtra := `{"content_id":"content123", "extra_field":"should_be_ignored", "content_type":"CONTENT_TYPE_POST"}`
		restored := &NexusContentObjectV1{}
		// protojson.UnmarshalOptions by default DiscardUnknown: false, which means it errors on unknown fields.
		// To ignore, set DiscardUnknown: true in FromJSON's unmarshaler options.
		// For now, expecting an error.
		err := restored.FromJSON(jsonWithExtra)
		assert.Error(t, err, "FromJSON() with extra fields should error by default")

		// If we wanted to allow unknown fields, we'd modify FromJSON, or use a local unmarshaler:
		// unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
		// err = unmarshaler.Unmarshal([]byte(jsonWithExtra), restored)
		// require.NoError(t, err)
		// assert.Equal(t, "content123", restored.ContentId)
	})
}

func TestNexusUserObjectV1_Serialization(t *testing.T) {
	t.Parallel()
	original := &NexusUserObjectV1{
		UserDid:      "did:example:user123",
		DisplayName:  "Test User",
		Bio:          "A user for testing.",
		AvatarUri:    "https://example.com/avatar.png",
		BannerUri:    "https://example.com/banner.png",
		CreatedAt:    1678886400000000000,
		UpdatedAt:    1678886401000000000,
		ProfileLinks: map[string]string{"website": "https://example.com", "twitter": "https://twitter.com/example"},
		IsVerified:   true,
	}

	t.Run("ToJSON_FromJSON_RoundTrip", func(t *testing.T) {
		t.Parallel()
		jsonString, err := original.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonString)

		assert.Contains(t, jsonString, `"user_did":`)
		assert.Contains(t, jsonString, `"is_verified":true`)

		restored := &NexusUserObjectV1{}
		err = restored.FromJSON(jsonString)
		require.NoError(t, err)
		assert.True(t, proto.Equal(original, restored))
	})
}

func TestNexusInteractionRecordV1_Serialization(t *testing.T) {
	t.Parallel()
	original := &NexusInteractionRecordV1{
		InteractionId:          "interaction789",
		UserDid:                "did:example:user123",
		TargetContentId:        "content123",
		InteractionType:        InteractionType_INTERACTION_TYPE_LIKE,
		InteractionData:        `{"reaction": "👍"}`,
		CreatedAtClient:        1678886402000000000,
		CreatedAtNetwork:       1678886403000000000,
		ReferenceInteractionId: "interactionParent456",
	}

	t.Run("ToJSON_FromJSON_RoundTrip", func(t *testing.T) {
		t.Parallel()
		jsonString, err := original.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonString)

		assert.Contains(t, jsonString, `"interaction_id":`)
		assert.Contains(t, jsonString, `"interaction_type":"INTERACTION_TYPE_LIKE"`)

		restored := &NexusInteractionRecordV1{}
		err = restored.FromJSON(jsonString)
		require.NoError(t, err)
		assert.True(t, proto.Equal(original, restored))
	})
}

func TestWitnessProofV1_Serialization(t *testing.T) {
	t.Parallel()
	original := &WitnessProofV1{
		ProofId:             "proofABC",
		WitnessDid:          "did:example:witness987",
		ObservedContentId:   "content123",
		ObservedContentHash: "sha256-...",
		ObservedAtWitness:   1678886404000000000,
		Signature:           "cryptographic_signature_here",
		Status:              WitnessProofStatus_WITNESS_PROOF_STATUS_VERIFIED,
		VerifiedAt:          1678886405000000000,
		VerifierDid:         "did:example:verifier654",
	}

	t.Run("ToJSON_FromJSON_RoundTrip", func(t *testing.T) {
		t.Parallel()
		jsonString, err := original.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonString)

		assert.Contains(t, jsonString, `"proof_id":`)
		assert.Contains(t, jsonString, `"status":"WITNESS_PROOF_STATUS_VERIFIED"`)

		restored := &WitnessProofV1{}
		err = restored.FromJSON(jsonString)
		require.NoError(t, err)
		assert.True(t, proto.Equal(original, restored))
	})
}

// Test standard library JSON marshalling compatibility (optional, but good for interop)
func TestStandardJSONMarshalling(t *testing.T) {
	t.Parallel()
	content := &NexusContentObjectV1{
		ContentId:   "contentStdJson",
		AuthorDid:   "did:example:std",
		ContentType: ContentType_CONTENT_TYPE_ARTICLE,
	}

	// Marshal using standard json.Marshal
	// This relies on the MarshalProtoMessageJSON function which internally calls our ToJSON.
	// To make this work directly, NexusContentObjectV1 would need to implement json.Marshaler.
	// For now, we test the helper.
	jsonData, err := MarshalProtoMessageJSON(content)
	require.NoError(t, err)

	var rawMap map[string]interface{}
	err = json.Unmarshal(jsonData, &rawMap)
	require.NoError(t, err)
	assert.Equal(t, "contentStdJson", rawMap["content_id"])
	assert.Equal(t, "CONTENT_TYPE_ARTICLE", rawMap["content_type"])

	// Unmarshal using standard json.Unmarshal
	// Similar to marshal, this tests the helper UnmarshalProtoMessageJSON.
	newContent := &NexusContentObjectV1{}
	err = UnmarshalProtoMessageJSON(jsonData, newContent)
	require.NoError(t, err)
	assert.True(t, proto.Equal(content, newContent))
}

// Test FromJSON with enum numbers (if UseEnumNumbers was true for marshalling)
// Our current ToJSON uses enum string values by default.
// FromJSON (protojson.UnmarshalOptions) by default accepts both enum names and numbers.
func TestFromJSON_EnumNumber(t *testing.T) {
	t.Parallel()
	// JSON string where enum is represented as a number
	jsonWithEnumNumber := `{"content_id":"content123", "content_type":1}` // 1 is CONTENT_TYPE_POST
	restored := &NexusContentObjectV1{}
	err := restored.FromJSON(jsonWithEnumNumber)
	require.NoError(t, err, "FromJSON() should accept enum numbers")
	assert.Equal(t, ContentType_CONTENT_TYPE_POST, restored.ContentType)
	assert.Equal(t, "content123", restored.ContentId)
}

func TestFromJSON_ErrorHandling(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		jsonString  string
		targetProto proto.Message
		expectError bool
	}{
		{
			name:        "NexusContentObjectV1 - valid",
			jsonString:  `{"content_id":"c1", "author_did":"d:a:1", "content_type":"CONTENT_TYPE_POST"}`,
			targetProto: &NexusContentObjectV1{},
			expectError: false,
		},
		{
			name:        "NexusContentObjectV1 - invalid JSON syntax",
			jsonString:  `{"content_id":"c1",`,
			targetProto: &NexusContentObjectV1{},
			expectError: true,
		},
		{
			name:        "NexusContentObjectV1 - type mismatch for field",
			jsonString:  `{"content_id": 123 }`, // content_id should be string
			targetProto: &NexusContentObjectV1{},
			expectError: true,
		},
		{
			name:        "NexusContentObjectV1 - unknown enum value string",
			jsonString:  `{"content_type":"NON_EXISTENT_TYPE"}`,
			targetProto: &NexusContentObjectV1{},
			expectError: true,
		},
		{
			name:        "NexusUserObjectV1 - valid",
			jsonString:  `{"user_did":"d:u:1", "display_name":"User"}`,
			targetProto: &NexusUserObjectV1{},
			expectError: false,
		},
		{
			name:        "NexusInteractionRecordV1 - valid",
			jsonString:  `{"interaction_id":"i1", "user_did":"d:u:1", "target_content_id":"c1", "interaction_type":"INTERACTION_TYPE_LIKE"}`,
			targetProto: &NexusInteractionRecordV1{},
			expectError: false,
		},
		{
			name:        "WitnessProofV1 - valid",
			jsonString:  `{"proof_id":"p1", "witness_did":"d:w:1", "observed_content_id":"c1", "status":"WITNESS_PROOF_STATUS_PENDING"}`,
			targetProto: &WitnessProofV1{},
			expectError: false,
		},
	}

	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := FromJSON(tc.jsonString, tc.targetProto)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test ToJSON options like EmitUnpopulated (currently false) and UseProtoNames (currently true)
func TestToJSON_Options(t *testing.T) {
	t.Parallel()
	obj := &NexusContentObjectV1{
		ContentId:   "test",
		ContentType: ContentType_CONTENT_TYPE_UNSPECIFIED, // Explicitly set to zero value
	}
	jsonString, err := obj.ToJSON()
	require.NoError(t, err)

	// Check for UseProtoNames (content_id vs contentId)
	assert.Contains(t, jsonString, `"content_id"`)
	assert.NotContains(t, jsonString, `"contentId"`)

	// Check for EmitUnpopulated (ContentType is unspecified, should it be emitted?)
	// With EmitUnpopulated: false, and UseEnumNumbers: false, zero value enums are emitted as their string name.
	// If it were a simple string or int field with zero value, it would be omitted.
	// With EmitUnpopulated: false, protojson's behavior for zero scalar values (like default enum or false bool)
	// is actually to OMIT them. The documentation can be a bit misleading.
	// "Scalar fields ... are emitted with their default values" might mean if they are part of a list/map,
	// but for top-level message fields, zero values are often omitted.
	assert.NotContains(t, jsonString, `"content_type":`, "Zero value enum string should be omitted with EmitUnpopulated:false")

	// An actually empty string field like AuthorDid should be omitted (as it's the zero value for string).
	assert.NotContains(t, jsonString, `"author_did":`)

	// Test with a boolean field
	userObj := &NexusUserObjectV1{
		UserDid:    "did:test:user",
		IsVerified: false, // Explicitly set to zero value
	}
	userJsonString, errUser := userObj.ToJSON() // Use different error variable name
	require.NoError(t, errUser)
	// Similarly, 'false' boolean values are omitted with EmitUnpopulated:false
	assert.NotContains(t, userJsonString, `"is_verified":`, "False boolean value should be omitted with EmitUnpopulated:false")
	t.Logf("User JSON for options test: %s", userJsonString)
}

func TestGenericSerializationHelpers(t *testing.T) {
	t.Parallel()
	original := &NexusContentObjectV1{ContentId: "c1", ContentType: ContentType_CONTENT_TYPE_POST}

	jsonData, err := MarshalProtoMessageJSON(original)
	require.NoError(t, err)

	var newObj NexusContentObjectV1
	err = UnmarshalProtoMessageJSON(jsonData, &newObj)
	require.NoError(t, err)

	assert.True(t, proto.Equal(original, &newObj))

	// Error case for Marshal
	// To cause an error in protojson.Marshal is tricky with valid inputs.
	// Usually, it involves an invalid message type or options.
	// For this test, we assume valid messages.

	// Error case for Unmarshal
	invalidJsonData := []byte(`{"content_id":123`) // Invalid JSON
	err = UnmarshalProtoMessageJSON(invalidJsonData, &newObj)
	assert.Error(t, err)
}
