package hardware

import (
	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/disk"
	"log"
	"strings"
)

func Get_all() ([]disk.PartitionStat) {
	devices,err:=disk.Partitions(false)
	var filter []disk.PartitionStat
	if err!=nil {
		log.Printf("get devices fail:%s",err)

	}
	for _,d:=range devices{
		if strings.Contains(d.Device,"loop") {
			continue
		}
		if strings.Contains(d.Device,"snap") {
			continue
		}
		filter=append(filter,d)
	}
	return filter
}

func Get_block() (*ghw.BlockInfo,error) {
	block,err:=ghw.Block()
	if err!=nil {
		log.Printf("get block devces fail: %s",err)
		return nil,err
	}
	//for _, disk2 := range block.Disks {
	//	fmt.Printf(" %v\n", disk2)
	//	for _, part := range disk2.Partitions {
	//		fmt.Printf("  %v\n", part)
	//	}
	//}
	return block,nil
}