package app

const (
	ProductName       = "FlashShell"
	LegacyProductName = "FlashDock"
)

func productMacOSBinaryNames() []string {
	return []string{ProductName, LegacyProductName}
}

func productAppBundleFileNames() []string {
	return []string{ProductName + ".app", LegacyProductName + ".app"}
}

func productWindowsExeNames() []string {
	return []string{ProductName + ".exe", LegacyProductName + ".exe"}
}
