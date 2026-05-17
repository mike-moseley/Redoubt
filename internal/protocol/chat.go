package protocol

import "github.com/google/uuid"

type MessageDestination int

const (
	DestinationLocal = iota
	DestinationDirect
	DestinationGlobal
	DestinationGroup
)

type Chat struct {
	Source uuid.UUID
	SourceName string
	Destination MessageDestination
	Contents string
}
