package main

// CraftState represents the Event data for updating the crafting window.
type CraftState struct {
	U1                uint32 // always 0?
	U3                uint32 // always 0?
	U4                uint32 // always 0?
	CraftAction       uint32 `json:"-"` // disable JSON so we can override it
	U2                uint32 // always 0?
	StepNum           uint32
	Progress          uint32
	ProgressDiff      int32
	Quality           uint32
	QualityDiff       int32
	HQChance          uint32
	Durability        uint32
	DurabilityDiff    int32
	CurrentCondition  uint32     `json:"-"` // disable JSON so we can override it
	PreviousCondition uint32     `json:"-"` // disable JSON so we can override it
	U6                [17]uint32 `json:"-"` // seems kinda random junk?
}

func (CraftState) isEventPlay32Data() {}

// Override the way Actions and Conditions are shown in JSON
func (c CraftState) MarshalJSON() ([]byte, error) {
	type Alias CraftState
	u6 := make([]int, len(c.U6))
	for i, b := range u6 {
		data[i] = int(b)
	}

	return json.Marshal(&struct {
		CraftAction       string     `json:"CraftAction"`
		CurrentCondition  string     `json:"CurrentCondition"`
		PreviousCondition string     `json:"PreviousCondition"`
		U6                [17]uint32 `json:"U6"`
		*Alias
	}{
		CraftAction:       actionName(c.CraftAction),
		CurrentCondition:  conditionName(c.CurrentCondition),
		PreviousCondition: conditionName(c.PreviousCondition),
		U6:                u6,
		Alias:             (*Alias)(c),
	})
}

func actionName(id uint32) string {
	return actionMap[id]
}

func conditionName(id uint32) string {
	return conditionMap[id]
}
