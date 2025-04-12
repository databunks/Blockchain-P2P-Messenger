package auth

var AllowedList = []string{}
var Administrator_Status = false

func AddToAllowedList(ip string) {
	AllowedList = append(AllowedList, ip)
}
