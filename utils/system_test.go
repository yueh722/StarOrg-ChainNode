package utils

import (
	"fmt"
	"testing"
)

func TestGetAllIPAddresses(t *testing.T) {

	got, err := GetAllIPAddresses()
	ip, err := FindIPByPrefix(got, "192.168.1")
	fmt.Println(ip, err)
}
