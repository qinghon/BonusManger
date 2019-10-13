package tools

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func RunCommand(cmd_str string) error {
	cmd := exec.Command("sh", "-c", cmd_str)
	log.Printf("sh -c %s", cmd_str)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

/*获取当前文件执行的路径*/
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])

	//得到全路径，比如在windows下E:\\golang\\test\\a.exe
	path, _ := filepath.Abs(file)

	rst := filepath.Dir(path)

	return rst
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

