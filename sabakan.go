package neco

import (
	"fmt"
)

// SystemContainers are fundamental containers not in artifacts
var SystemContainers = []ContainerImage{
	{
		Name:       "bird",
		Repository: "quay.io/cybozu/bird",
		Tag:        "2.0",
	},
	{
		Name:       "chrony",
		Repository: "quay.io/cybozu/chrony",
		Tag:        "3.3",
	},
	{
		Name:       "ubuntu-debug",
		Repository: "quay.io/cybozu/ubuntu-debug",
		Tag:        "18.04",
	},
}

// SystemImagePath returns path of ACI file for the system container img
func SystemImagePath(img ContainerImage) string {
	return fmt.Sprintf("/extras/cybozu-%s-%s.aci", img.Name, img.Tag)
}

// ImageAssetName returns asset's name for the img
func ImageAssetName(img ContainerImage) string {
	return fmt.Sprintf("cybozu-%s-%s.img", img.Name, img.Tag)
}

// ACIAssetName returns aci image file name of img
func ACIAssetName(img ContainerImage) string {
	return fmt.Sprintf("cybozu-%s-%s.aci", img.Name, img.Tag)
}

// CryptsetupAssetName returns asset's name for sabakan-cryptsetup
func CryptsetupAssetName(img ContainerImage) string {
	return "sabakan-cryptsetup-" + img.Tag
}
