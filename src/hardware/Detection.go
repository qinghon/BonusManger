package hardware

import (
	"bufio"
	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/disk"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Get_all() ([]disk.PartitionStat) {
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

func Get_block() (*ghw.BlockInfo, error) {
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
func CreatePart(dev, tp, begin, end string) ([]byte, error) {
	// tp: "aix", "amiga", "bsd", "dvh", "gpt", "loop", "mac", "msdos", "pc98", or "sun"
	cmd := exec.Command("parted", dev, "mkpart", tp, begin, end)
	return cmd.Output()
}
func DeletePart(dev ,part string)([]byte, error) {
	cmd := exec.Command("parted", dev, "rm", part)
	return cmd.Output()
}
