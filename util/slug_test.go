package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SanitizeGroupSlug", func() {
	It("should lowercase and replace spaces with hyphens", func() {
		Expect(SanitizeGroupSlug("Kids Devices")).To(Equal("kids-devices"))
	})

	It("should replace underscores with hyphens", func() {
		Expect(SanitizeGroupSlug("IoT_Network")).To(Equal("iot-network"))
	})

	It("should replace dots with hyphens", func() {
		Expect(SanitizeGroupSlug("Guest WiFi 2.4")).To(Equal("guest-wifi-2-4"))
	})

	It("should strip non-alphanumeric/non-hyphen characters", func() {
		Expect(SanitizeGroupSlug("kids@home!")).To(Equal("kidshome"))
	})

	It("should collapse multiple hyphens", func() {
		Expect(SanitizeGroupSlug("kids---devices")).To(Equal("kids-devices"))
	})

	It("should trim leading and trailing hyphens", func() {
		Expect(SanitizeGroupSlug("-kids-")).To(Equal("kids"))
	})

	It("should handle already-valid slugs", func() {
		Expect(SanitizeGroupSlug("kids-devices")).To(Equal("kids-devices"))
	})

	It("should handle empty string", func() {
		Expect(SanitizeGroupSlug("")).To(Equal(""))
	})

	It("should handle whitespace-only string", func() {
		Expect(SanitizeGroupSlug("   ")).To(Equal(""))
	})

	It("should truncate to 63 characters", func() {
		long := "abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklm"
		result := SanitizeGroupSlug(long)
		Expect(len(result)).To(BeNumerically("<=", 63))
	})

	It("should not end with a hyphen after truncation", func() {
		// 63 chars where the 63rd is a hyphen
		input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbb"
		result := SanitizeGroupSlug(input)
		Expect(result).NotTo(HaveSuffix("-"))
		Expect(len(result)).To(BeNumerically("<=", 63))
	})

	It("should handle unicode by stripping non-ASCII", func() {
		Expect(SanitizeGroupSlug("café-réseau")).To(Equal("caf-rseau"))
	})

	It("should handle mixed separators", func() {
		Expect(SanitizeGroupSlug("my_cool device.v2")).To(Equal("my-cool-device-v2"))
	})
})
