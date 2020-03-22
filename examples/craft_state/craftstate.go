package main

// CraftState represents the Event data for updating the crafting window.
type CraftState struct {
	U1                uint32
	U3                uint32
	U4                uint32
	CraftAction       uint32
	U2                uint32
	StepNum           uint32
	Progress          uint32
	ProgressDiff      int32
	Quality           uint32
	QualityDiff       int32
	HQChance          uint32
	Durability        uint32
	DurabilityDiff    int32
	CurrentCondition  uint32
	PreviousCondition uint32
	U6                [17]uint32
}

func (CraftState) isEventPlay32Data() {}
