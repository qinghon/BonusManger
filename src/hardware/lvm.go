package hardware

import (
	"encoding/json"
	"fmt"
	"github.com/qinghon/system/package"
	"os/exec"
	"syscall"
)

type PV struct {
	Report []struct {
		Pv []Pv `json:"pv"`
	} `json:"report"`
}
type Pv struct {
	PvName string `json:"pv_name"`
	VgName string `json:"vg_name"`
	PvFmt  string `json:"pv_fmt"`
	PvAttr string `json:"pv_attr"`
	PvSize string `json:"pv_size"`
	PvFree string `json:"pv_free"`
}

type VG struct {
	Report []struct {
		Vg []Vg `json:"vg"`
	} `json:"report"`
}
type Vg struct {
	VgName    string `json:"vg_name"`
	PvCount   string `json:"pv_count"`
	LvCount   string `json:"lv_count"`
	SnapCount string `json:"snap_count"`
	VgAttr    string `json:"vg_attr"`
	VgSize    string `json:"vg_size"`
	VgFree    string `json:"vg_free"`
}

type LV struct {
	Report []struct {
		Lv []Lv `json:"lv"`
	} `json:"report"`
}
type Lv struct {
	LvName          string `json:"lv_name"`
	VgName          string `json:"vg_name"`
	LvAttr          string `json:"lv_attr"`
	LvSize          string `json:"lv_size"`
	PoolLv          string `json:"pool_lv"`
	Origin          string `json:"origin"`
	DataPercent     string `json:"data_percent"`
	MetadataPercent string `json:"metadata_percent"`
	MovePv          string `json:"move_pv"`
	MirrorLog       string `json:"mirror_log"`
	CopyPercent     string `json:"copy_percent"`
	ConvertLv       string `json:"convert_lv"`
}

func install_lvm() ([]byte, error) {
	if !_package.CheckExec("lvs") {
		return _package.Install("lvm2")
	} else {
		return nil, nil
	}
}
func GetPv() (PV, error) {
	install_lvm()
	cmd := exec.Command("pvs", "--reportformat", "json")
	out, err := cmd.Output()
	if err != nil {
		return PV{}, err
	}
	var pv PV
	err = json.Unmarshal(out, &pv)
	if err != nil {
		return PV{}, err
	} else {
		return pv, nil
	}
}

func CreatePV(dev string) (PV, error) {
	cmd := exec.Command("pvcreate", dev)
	_, err := cmd.Output()
	if err != nil {
		return PV{}, err
	}
	return GetPv()
}

func RemovePV(pv Pv) (PV, error) {
	if pv.VgName != "" {
		if _, err := RemoveVg(Vg{pv.VgName, "", "", "", "", "", ""}); err != nil {
			return PV{}, err
		}
	}
	cmd := exec.Command("pvremove", pv.PvName)
	_, err := cmd.Output()
	if err != nil {
		return PV{}, err
	}
	return GetPv()
}

func GetVg() (VG, error) {
	install_lvm()
	cmd := exec.Command("vgs", "--reportformat", "json")
	out, err := cmd.Output()
	if err != nil {
		return VG{}, err
	}
	var vg VG
	err = json.Unmarshal(out, &vg)
	if err != nil {
		return VG{}, err
	} else {
		return vg, nil
	}
}

func CreateVg(VgName string, dev ...string) (VG, error) {
	tmp := append([]string{VgName}, dev...)
	cmd := exec.Command("vgcreate", tmp...)
	_, err := cmd.Output()
	if err != nil {
		return VG{}, err
	}
	return GetVg()
}

func RemoveVg(vg Vg) (VG, error) {
	lvs, err := GetLv() //clean lv begain
	if err != nil {
		return VG{}, err
	}
	var tmp []Lv
	for _, lv := range lvs.Report[0].Lv {
		if lv.VgName == vg.VgName {
			tmp = append(tmp, lv)
		}
	}
	_, err = RemoveLv(tmp)
	if err != nil {
		return VG{}, err
	}
	// clean lv end
	// clean vg begain
	cmd := exec.Command("vgremove", vg.VgName)
	_, err = cmd.Output()
	if err != nil {
		return VG{}, err
	}
	return GetVg()
}
func GetLv() (LV, error) {
	install_lvm()
	cmd := exec.Command("lvs", "--reportformat", "json")
	out, err := cmd.Output()
	if err != nil {
		return LV{}, err
	}
	var lv LV
	err = json.Unmarshal(out, &lv)
	if err != nil {
		return LV{}, err
	} else {
		return lv, nil
	}
}

func CreateLv(lv Lv) (LV, error) {
	cmd := exec.Command("lvcreate", "-L", lv.LvSize, "-n", lv.LvName, lv.VgName)
	_, err := cmd.Output()
	if err != nil {
		return LV{}, err
	}
	return GetLv()
}

func RemoveLv(lv []Lv) (LV, error) {
	mts, err := ReadMounts("/proc/mounts")
	if err != nil {
		return LV{}, err
	}

	for _, l := range lv {
		for _, m := range mts.Mounts {
			if fmt.Sprintf("/dev/%s-%s", l.VgName, l.LvName) == m.Device {
				if err := Umount(m.MountPoint); err != nil {
					return LV{}, err
				}
			}
		}
		cmd := exec.Command("lvremove", fmt.Sprintf("/dev/%s/%s", l.VgName, l.LvName))
		_, err := cmd.Output()
		if err != nil {
			return LV{}, err
		}
	}
	return GetLv()
}

func Umount(dev string) error {
	// dev: /mnt is mounted point
	return syscall.Unmount(dev, 0)
}
func UmountDev(dev string) error {
	mts, err := ReadMounts("/proc/mounts")
	if err != nil {
		return err
	}
	for _, m := range mts.Mounts {
		if fmt.Sprintf("/dev/%s", dev) == m.Device {
			if err := Umount(m.MountPoint); err != nil {
				return err
			}
		}
	}
	return nil
}
