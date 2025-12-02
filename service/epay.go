package service

import (
	"xunkecloudAPI/setting/operation_setting"
	"xunkecloudAPI/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
