package utils

import (
	"fmt"
	"strings"
)

// ClientInfo 客户端信息结构
type ClientInfo struct {
	UserAgent string
	IPAddress string
}

// ParseClientInfo 解析客户端信息字符串
// 格式: user_agent|ip_address
func ParseClientInfo(clientInfoStr string) ClientInfo {
	if clientInfoStr == "" {
		return ClientInfo{UserAgent: "unknown", IPAddress: "unknown"}
	}

	parts := strings.Split(clientInfoStr, "|")
	if len(parts) != 2 {
		return ClientInfo{UserAgent: "unknown", IPAddress: "unknown"}
	}

	return ClientInfo{
		UserAgent: parts[0],
		IPAddress: parts[1],
	}
}

// FormatClientInfo 格式化客户端信息为字符串
// 格式: user_agent|ip_address
func FormatClientInfo(userAgent, ipAddress string) string {
	if userAgent == "" {
		userAgent = "unknown"
	}
	if ipAddress == "" {
		ipAddress = "unknown"
	}
	return fmt.Sprintf("%s|%s", userAgent, ipAddress)
}

// LocationInfo 地理位置信息结构
type LocationInfo struct {
	Country  string
	Province string
	City     string
}

// ParseLocationInfo 解析地理位置信息字符串
// 格式: 国家|省份|城市
func ParseLocationInfo(locationInfoStr string) LocationInfo {
	if locationInfoStr == "" {
		return LocationInfo{Country: "unknown", Province: "unknown", City: "unknown"}
	}

	parts := strings.Split(locationInfoStr, "|")
	if len(parts) != 3 {
		return LocationInfo{Country: "unknown", Province: "unknown", City: "unknown"}
	}

	return LocationInfo{
		Country:  parts[0],
		Province: parts[1],
		City:     parts[2],
	}
}

// FormatLocationInfo 格式化地理位置信息为字符串
// 格式: 国家|省份|城市
func FormatLocationInfo(country, province, city string) string {
	if country == "" {
		country = "unknown"
	}
	if province == "" {
		province = "unknown"
	}
	if city == "" {
		city = "unknown"
	}
	return fmt.Sprintf("%s|%s|%s", country, province, city)
}

// DeviceInfo 设备信息结构
type DeviceInfo struct {
	OS      string
	Browser string
	Device  string
}

// ParseDeviceInfo 解析设备信息字符串
// 格式: 操作系统|浏览器|设备类型
func ParseDeviceInfo(deviceInfoStr string) DeviceInfo {
	if deviceInfoStr == "" {
		return DeviceInfo{OS: "unknown", Browser: "unknown", Device: "unknown"}
	}

	parts := strings.Split(deviceInfoStr, "|")
	if len(parts) != 3 {
		return DeviceInfo{OS: "unknown", Browser: "unknown", Device: "unknown"}
	}

	return DeviceInfo{
		OS:      parts[0],
		Browser: parts[1],
		Device:  parts[2],
	}
}

// FormatDeviceInfo 格式化设备信息为字符串
// 格式: 操作系统|浏览器|设备类型
func FormatDeviceInfo(os, browser, device string) string {
	if os == "" {
		os = "unknown"
	}
	if browser == "" {
		browser = "unknown"
	}
	if device == "" {
		device = "unknown"
	}
	return fmt.Sprintf("%s|%s|%s", os, browser, device)
}
