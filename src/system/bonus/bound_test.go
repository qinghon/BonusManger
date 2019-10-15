package bonus

import "testing"

func Test_ReadClientBcode(t *testing.T) {
	ReadClientBcode("/opt/bcloud_bak/client.crt")
}
func TestReadBcode(t *testing.T) {
	bcode, err := ReadBcode()
	if err != nil {
		t.Error(err)
	}
	t.Log(bcode)
}
