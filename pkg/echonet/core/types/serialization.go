package types

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ToJSON marshals a proto.Message to a JSON string.
func ToJSON(m proto.Message) (string, error) {
	marshaler := protojson.MarshalOptions{
		Multiline:       false,
		Indent:          "",
		AllowPartial:    false, // Set to true if you want to allow partial messages
		UseProtoNames:   true,  // Use original protobuf field names
		UseEnumNumbers:  false, // Use enum string values
		EmitUnpopulated: false, // Do not emit fields with zero values
	}
	b, err := marshaler.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FromJSON unmarshals a JSON string to a proto.Message.
func FromJSON(jsonString string, m proto.Message) error {
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   false, // Set to true if you want to allow partial messages
		DiscardUnknown: false, // Set to true to discard unknown fields
	}
	return unmarshaler.Unmarshal([]byte(jsonString), m)
}

// --- NexusContentObjectV1 ---

// ToJSON converts NexusContentObjectV1 to its JSON string representation.
func (x *NexusContentObjectV1) ToJSON() (string, error) {
	return ToJSON(x)
}

// FromJSON populates NexusContentObjectV1 from its JSON string representation.
func (x *NexusContentObjectV1) FromJSON(jsonString string) error {
	return FromJSON(jsonString, x)
}

// --- NexusUserObjectV1 ---

// ToJSON converts NexusUserObjectV1 to its JSON string representation.
func (x *NexusUserObjectV1) ToJSON() (string, error) {
	return ToJSON(x)
}

// FromJSON populates NexusUserObjectV1 from its JSON string representation.
func (x *NexusUserObjectV1) FromJSON(jsonString string) error {
	return FromJSON(jsonString, x)
}

// --- NexusInteractionRecordV1 ---

// ToJSON converts NexusInteractionRecordV1 to its JSON string representation.
func (x *NexusInteractionRecordV1) ToJSON() (string, error) {
	return ToJSON(x)
}

// FromJSON populates NexusInteractionRecordV1 from its JSON string representation.
func (x *NexusInteractionRecordV1) FromJSON(jsonString string) error {
	return FromJSON(jsonString, x)
}

// --- WitnessProofV1 ---

// ToJSON converts WitnessProofV1 to its JSON string representation.
func (x *WitnessProofV1) ToJSON() (string, error) {
	return ToJSON(x)
}

// FromJSON populates WitnessProofV1 from its JSON string representation.
func (x *WitnessProofV1) FromJSON(jsonString string) error {
	return FromJSON(jsonString, x)
}

// Generic helper for cases where we might deal with the raw json.RawMessage
// This is not strictly required by the current task but can be useful.

// MarshalJSON implements the json.Marshaler interface for proto.Message.
// This allows direct use of proto messages with the standard json.Marshal.
func MarshalProtoMessageJSON(m proto.Message) ([]byte, error) {
	s, err := ToJSON(m)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalProtoMessageJSON implements the json.Unmarshaler interface for proto.Message.
// This allows direct use of proto messages with the standard json.Unmarshal.
func UnmarshalProtoMessageJSON(data []byte, m proto.Message) error {
	return FromJSON(string(data), m)
}
