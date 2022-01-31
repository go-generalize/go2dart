package standard

import "time"

type StatusT string

const (
	StatusOK      StatusT = "OK"
	StatusFailure StatusT = "Failure"
)

type ModeT int

const (
	Enabled  ModeT = 1
	Disabled ModeT = 2
)

type StructInStruct struct {
	A bool
	B string
	C []string
}

type PostUserRequest struct {
	T          time.Time
	TPtr       *time.Time
	TPtrNull   *time.Time
	S          string `json:"s"`
	Status     map[string][]StatusT
	Mode       *ModeT
	Array      []string
	B          map[string]bool
	SinS       StructInStruct
	SinSs      []StructInStruct
	DynamicMap map[string]interface{}
}

type EmptyStruct struct {
}
