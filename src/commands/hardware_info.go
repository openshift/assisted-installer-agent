package commands

func GetHardwareInfo(_ string, _ []string) (string, string, int) {
	return string(CreateHostInfo()) ,"",  0
}
