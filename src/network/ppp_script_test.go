package network

import "testing"

func TestSetAllAuto(t *testing.T) {
	err := SetAllAuto()
	if err != nil {
		t.Error(err)
	}
}
