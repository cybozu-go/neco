package neco

import (
	"fmt"
)

// SystemContainers are fundamental containers not in artifacts
var SystemContainers = []ContainerImage{
	{
		Name:       "bird",
		Repository: "quay.io/cybozu/bird",
		Tag:        "2.0.2-7",
	},
	{
		Name:       "chrony",
		Repository: "quay.io/cybozu/chrony",
		Tag:        "3.3-4",
	},
}

// ImageAssetName returns asset's name for the img
func ImageAssetName(img ContainerImage) string {
	return fmt.Sprintf("cybozu-%s-%s.aci", img.Name, img.Tag)
}

// CryptsetupAssetName returns asset's name for sabakan-cryptsetup
func CryptsetupAssetName(version string) string {
	return "sabakan-cryptsetup-" + version
}
