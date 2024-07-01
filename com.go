package emit

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// 获取随机ip
func RandIP() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ip2Int := func(ip string) int {
		slice := strings.Split(ip, ".")
		result := 0
		i, _ := strconv.Atoi(slice[0])
		result += i << 24
		i, _ = strconv.Atoi(slice[1])
		result += i << 16
		i, _ = strconv.Atoi(slice[2])
		result += i << 8
		i, _ = strconv.Atoi(slice[3])
		result += i
		return result
	}

	int2Ip := func(num int) (result string) {
		result += strconv.Itoa(num>>24&255) + "."
		result += strconv.Itoa(num>>16&255) + "."
		result += strconv.Itoa(num>>8&255) + "."
		result += strconv.Itoa(num & 255)
		return
	}
	randIndex := r.Intn(len(_range))
	startIPInt := ip2Int(_range[randIndex][0])
	endIPInt := ip2Int(_range[randIndex][1])

	newIpInt := r.Intn(endIPInt-startIPInt) + startIPInt
	return int2Ip(newIpInt)
}

var _range = [][]string{
	{"4.150.64.0", "4.150.127.255"},      // Azure Cloud EastUS2 16382
	{"4.152.0.0", "4.153.255.255"},       // Azure Cloud EastUS2 131070
	{"13.68.0.0", "13.68.127.255"},       // Azure Cloud EastUS2 32766
	{"13.104.216.0", "13.104.216.255"},   // Azure EastUS2 256
	{"20.1.128.0", "20.1.255.255"},       // Azure Cloud EastUS2 32766
	{"20.7.0.0", "20.7.255.255"},         // Azure Cloud EastUS2 65534
	{"20.22.0.0", "20.22.255.255"},       // Azure Cloud EastUS2 65534
	{"40.84.0.0", "40.84.127.255"},       // Azure Cloud EastUS2 32766
	{"40.123.0.0", "40.123.127.255"},     // Azure Cloud EastUS2 32766
	{"4.214.0.0", "4.215.255.255"},       // Azure Cloud JapanEast 131070
	{"4.241.0.0", "4.241.255.255"},       // Azure Cloud JapanEast 65534
	{"40.115.128.0", "40.115.255.255"},   // Azure Cloud JapanEast 32766
	{"52.140.192.0", "52.140.255.255"},   // Azure Cloud JapanEast 16382
	{"104.41.160.0", "104.41.191.255"},   // Azure Cloud JapanEast 8190
	{"138.91.0.0", "138.91.15.255"},      // Azure Cloud JapanEast 4094
	{"151.206.65.0", "151.206.79.255"},   // Azure Cloud JapanEast 256
	{"191.237.240.0", "191.237.241.255"}, // Azure Cloud JapanEast 512
	{"4.208.0.0", "4.209.255.255"},       // Azure Cloud NorthEurope 131070
	{"52.169.0.0", "52.169.255.255"},     // Azure Cloud NorthEurope 65534
	{"68.219.0.0", "68.219.127.255"},     // Azure Cloud NorthEurope 32766
	{"65.52.64.0", "65.52.79.255"},       // Azure Cloud NorthEurope 4094
	{"98.71.0.0", "98.71.127.255"},       // Azure Cloud NorthEurope 32766
	{"74.234.0.0", "74.234.127.255"},     // Azure Cloud NorthEurope 32766
	{"4.151.0.0", "4.151.255.255"},       // Azure Cloud SouthCentralUS 65534
	{"13.84.0.0", "13.85.255.255"},       // Azure Cloud SouthCentralUS 131070
	{"4.255.128.0", "4.255.255.255"},     // Azure Cloud WestCentralUS 32766
	{"13.78.128.0", "13.78.255.255"},     // Azure Cloud WestCentralUS 32766
	{"4.175.0.0", "4.175.255.255"},       // Azure Cloud WestEurope 65534
	{"13.80.0.0", "13.81.255.255"},       // Azure Cloud WestEurope 131070
	{"20.73.0.0", "20.73.255.255"},       // Azure Cloud WestEurope 65534
}

func IsEncoding(data []byte, encoding string) bool {
	switch encoding {
	case "gzip":
		return len(data) >= 2 &&
			data[0] == 0x1F &&
			data[1] == 0x8B
	case "deflate":
		// ？？？
		return true
	case "br":
		return len(data) >= 3 &&
			data[0] == 0xCE &&
			data[1] == 0xB2 &&
			data[2] == 0xCF
	default:
		return false
	}
}
