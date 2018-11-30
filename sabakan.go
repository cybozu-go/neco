package neco

import (
	"fmt"
)

// ImageAssetName returns asset's name for the img
func ImageAssetName(img ContainerImage) string {
	return fmt.Sprintf("cybozu-%s-%s.aci", img.Name, img.Tag)
}

// CryptsetupAssetName returns asset's name for sabakan-cryptsetup
func CryptsetupAssetName(version string) string {
	return "sabakan-cryptsetup-" + version
}
