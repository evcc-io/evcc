package api

// BytesMarshaler marshals into bytes/string representation
type BytesMarshaler interface {
	MarshalBytes() ([]byte, error)
}

// BytesMarshaler marshals into a struct representation
type StructMarshaler interface {
	MarshalStruct() (any, error)
}
