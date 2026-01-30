package utils

import "fmt"

func ConvertBytesToReadable[T int64 | uint64](size T) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TB", float64(size)/(1024*1024*1024*1024))
	}
}
