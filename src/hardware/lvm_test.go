package hardware

import "testing"

func Test_Umount(t *testing.T) {
	err:=Umount("/tmp/loop24")
	if err!=nil {
		t.Error(err)
	}
}
