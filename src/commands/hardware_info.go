package commands

func GetHardwareInfo(_ string, _ ...string) (stdout string, stderr string, exitCode int) {
	return string(CreateHostInfo()) ,"",  0
}
