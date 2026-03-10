package engine

import "testing"

func TestRepairJSON_PreservesTopLevelArray(t *testing.T) {
	input := "[{\"action\":\"Push\",\"argv\":[\"git\",\"push\"]}]"
	repaired := repairJSON(input)
	if repaired == "" || repaired[0] != '[' {
		t.Fatalf("expected repaired JSON to remain an array, got %q", repaired)
	}
}
