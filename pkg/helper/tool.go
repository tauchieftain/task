package helper

import (
	"fmt"
	"github.com/gofrs/uuid"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func FileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func UUID() string {
	uu, err := uuid.NewGen().NewV1()
	if err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return uu.String()
}

func IsExistInUintSlice(s []uint, item uint) bool {
	for _, i := range s {
		if i == item {
			return true
		}
	}
	return false
}

func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func OpenFile(path string, flag int) (*os.File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(absPath, flag, 0644)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(absPath), 0755)
		if err != nil {
			return nil, err
		}
		return os.OpenFile(absPath, flag, 0644)
	}
	return f, err
}

func HumanTime(s int64) string {
	var ht string
	day := s / (24 * 3600)
	if day > 0 {
		ht += strconv.Itoa(int(day)) + "天"
	}
	hour := (s - day*3600*24) / 3600
	if ht != "" || hour > 0 {
		ht += strconv.Itoa(int(hour)) + "时"
	}
	minute := (s - day*24*3600 - hour*3600) / 60
	if ht != "" || minute > 0 {
		ht += strconv.Itoa(int(minute)) + "分"
	}
	second := s - day*24*3600 - hour*3600 - minute*60
	if ht != "" || second > 0 {
		ht += strconv.Itoa(int(second)) + "秒"
	}
	return ht
}
