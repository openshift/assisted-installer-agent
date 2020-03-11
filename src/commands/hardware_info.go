package commands


func GetHardwareInfo(_ string) (string, error) {
	return string(CreateNodeInfo()), nil
}