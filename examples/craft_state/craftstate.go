package main

import (
	"encoding/json"
	"strconv"
)

// CraftState represents the Event data for updating the crafting window.
type CraftState struct {
	U1                uint32 `json:"-"` // junk from memcpy of other packets
	U3                uint32 `json:"-"` // junk from memcpy of other packets
	U4                uint32 `json:"-"` // junk from memcpy of other packets
	ActionID          uint32 `json:"-"` // disable JSON so we can override it
	U2                uint32 `json:"-"` // junk from memcpy of other packets
	Step              uint32
	Progress          uint32
	ProgressDiff      int32
	Quality           uint32
	QualityDiff       int32
	HQChance          uint32
	Durability        uint32
	DurabilityDiff    int32
	CurrentCondition  uint32     `json:"-"` // disable JSON so we can override it
	PreviousCondition uint32     `json:"-"` // disable JSON so we can override it
	U6                [17]uint32 `json:"-"` // junk from memcpy of other packets
}

func (CraftState) isEventPlay32Data() {}

// Override the way Actions and Conditions are shown in JSON
func (c CraftState) MarshalJSON() ([]byte, error) {
	type Alias CraftState

	return json.Marshal(&struct {
		ActionID          string `json:"ActionID"`
		CurrentCondition  string `json:"CurrentCondition"`
		PreviousCondition string `json:"PreviousCondition"`
		Alias
	}{
		ActionID:          actionName(c.ActionID),
		CurrentCondition:  conditionName(c.CurrentCondition),
		PreviousCondition: conditionName(c.PreviousCondition),
		Alias:             (Alias)(c),
	})
}

func actionName(id uint32) string {
	action := actionMap[id]
	if action == "" { action = strconv.Itoa(int(id)) }

	return action
}

func conditionName(id uint32) string {
	condition := conditionMap[id]
	if condition == "" { condition = strconv.Itoa(int(id)) }

	return condition
}
