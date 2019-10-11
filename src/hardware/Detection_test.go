package hardware

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_get_all(t *testing.T) {
	devices := Get_all()

	t.Log(devices)
}
func Test_get_block(t *testing.T) {
	Get_block()
}
func Test_ReadMounts(t *testing.T) {
	mo, err := ReadMounts("/proc/mounts")
	if err != nil {
		t.Error(err)
	}
	js, _ := json.Marshal(mo)
	fmt.Println(string(js))
}
