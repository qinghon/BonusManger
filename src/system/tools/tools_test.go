package tools

import "testing"

func TestReadHostKey(t *testing.T) {
	keys, err := ReadHostKey("wusheng")
	if err != nil {
		t.Error(err)
	}
	t.Log(keys)
}
func TestKey_IsTrust(t *testing.T) {
	keys, _ := ReadHostKey("wusheng")
	for _, k := range keys {
		if k.IsTrust() {
			t.Log(k.PublicKey)
		} else {
			t.Log(false, k.PublicKey)
		}
		//t.Log(k.IsTrust())
	}
}
