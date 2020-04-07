package hardware

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_get_all(t *testing.T) {
	devices := GetAll()

	t.Log(devices)
}
func Test_get_block(t *testing.T) {
	GetBlock()
}
func Test_ReadMounts(t *testing.T) {
	mo, err := ReadMounts("/proc/mounts")
	if err != nil {
		t.Error(err)
	}
	js, _ := json.Marshal(mo)
	fmt.Println(string(js))
}
func Test_Mount(t *testing.T) {
	p := Partition{"sdc1", "", "", "ext4", ""}
	t.Log(p.Mount("/tmp/sdc1"))
}
