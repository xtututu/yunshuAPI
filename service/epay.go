package service

import (
	"yishangyunApi/setting/operation_setting"
	"yishangyunApi/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
