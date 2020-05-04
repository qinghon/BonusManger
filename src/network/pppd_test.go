package network

import "testing"

func TestGetPPPDVersion(t *testing.T) {
	version, err := GetPPPDVersion()
	if err != nil {
		t.Error(err)
	}
	t.Log(version, len(version))
}
