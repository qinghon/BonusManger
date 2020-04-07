package hardware

import (
	"bufio"
	"github.com/jaypipes/ghw"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func GetAll() []disk.PartitionStat {
	devices, err := disk.Partitions(false)
	var filter []disk.PartitionStat
	if err != nil {
		log.Printf("get devices fail:%s", err)

	}
	for _, d := range devices {
		if strings.Contains(d.Device, "loop") {
			continue
		}
		if strings.Contains(d.Device, "snap") {
			continue
		}
		filter = append(filter, d)
	}
	return filter
}

func GetBlock() (*ghw.BlockInfo, error) {
	block, err := ghw.Block()
	if err != nil {
		log.Printf("get block devces fail: %s", err)
		return nil, err
	}
	//for _, disk2 := range block.Disks {
	//	fmt.Printf(" %v\n", disk2)
	//	for _, part := range disk2.Partitions {
	//		fmt.Printf("  %v\n", part)
	//	}
	//}
	return block, nil
}

type Mounts struct {
	Mounts []Mount `json:"mounts"`
}

type Mount struct {
	Device     string `json:"device"`
	MountPoint string `json:"mountpoint"`
	FSType     string `json:"fstype"`
	Options    string `json:"options"`
}

type Dev struct {
	ghw.Disk
	Table string `json:"table"`
}
type Partition struct {
	//Dev        *Dev   `json:"-"`
	Name       string `json:"name" binding:"required"`
	Begin      string `json:"begin" binding:"required"`
	End        string `json:"end" binding:"required"`
	FileSystem string `json:"file_system" binding:"required"`
	Label      string `json:"label"`
}

func ReadMounts(path string) (*Mounts, error) {
	fin, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	var mounts = Mounts{}

	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		var mount = &Mount{
			fields[0],
			fields[1],
			fields[2],
			fields[3],
		}
		mounts.Mounts = append(mounts.Mounts, *mount)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &mounts, nil
}

func (d *Dev) CreatePart(p Partition) ([]byte, error) {
	switch d.Table {
	case "msdos":
		return d.Run("mkpart", d.Table, p.FileSystem, p.Begin, p.End)
	case "gpt":
		return d.Run("mkpart", p.FileSystem, p.Begin, p.End)
	default:
		return nil, errors.New("not set device part table type")
	}
}
func (d *Dev) DeletePart(part string) ([]byte, error) {

	return d.Run("rm", part)
}
func (d *Dev) Mklabel(tp string) ([]byte, error) {
	// tp: "aix", "amiga", "bsd", "dvh", "gpt", "loop", "mac", "msdos", "pc98", or "sun"
	by, err := d.Run("mklabel", tp)
	if err != nil {
		return by, err
	}
	d.Table = tp
	return by, nil
}
func (d *Dev) Run(str ...string) ([]byte, error) {

	var tmp0 []string
	tmp0 = append(tmp0, "-s")
	tmp0 = append(tmp0, "/dev/"+d.Name)
	tmp0 = append(tmp0, str...)
	log.Println("parted ", tmp0)
	cmd := exec.Command("parted", tmp0...)
	return cmd.Output()
}
func DiskInfo(name string) (Dev, error) {
	var d Dev
	var da *ghw.Disk
	bk, err := ghw.Block()

	disks := bk.Disks
	for _, dk := range disks {
		if dk.Name == name {
			da = dk
			d.Name = name
		}
	}
	if d.Name == "" {
		return Dev{}, errors.New("device not found")
	}
	by, err := d.Run("print", "-m")
	if err != nil {
		log.Printf("get disk info fail:%s", err)
		return Dev{}, err
	}
	log.Println(string(by))
	newstr := strings.ReplaceAll(string(by), "\n", "")
	lines := strings.Split(newstr, ";")
	if len(lines) <= 1 {
		return Dev{}, errors.New("not found part table type")
	}
	log.Print(lines)
	table := strings.Split(lines[1], ":")[5]
	log.Println(strings.Split(lines[1], ":"))
	log.Println(table)
	d = Dev{*da, table}
	log.Println(d.Table)
	return d, nil
}

func (p Partition) Format(tp string) ([]byte, error) {
	// tp: ext4 , ext3 , ext2 , ntfs
	cmd := exec.Command("mkfs", "--type", tp, p.Name)
	by, err := cmd.Output()
	if err != nil {
		return by, err
	}
	p.FileSystem = tp
	return by, err
}

func (p Partition) Mount(_path string) error {
	return syscall.Mount("/dev/"+p.Name, _path, p.FileSystem, 0, "rw")
}
