package hardware

import (
	"testing"
)

func Test_get_all(t *testing.T)  {
	devices:= Get_all()

	t.Log(devices)
}
func Test_get_block(t *testing.T)  {
	Get_block()
}