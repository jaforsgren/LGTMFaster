package common

import "github.com/google/uuid"

func GetString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func GetBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func GetInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func GetUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
