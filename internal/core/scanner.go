package core

// RunScan executes the scanning logic for a given target and asset.
// It returns a channel that could push updates, or just runs in background.
func RunScan(targetInput string, assetName string) {
	GetManager().StartScan(targetInput, assetName)
}
