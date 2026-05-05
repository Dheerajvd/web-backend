package domain

import "strings"

// UserCanAccessApp mirrors login/token scoping: super users may use any app; others must have appId in appIds.
func UserCanAccessApp(role Role, appIDs []string, appID string) bool {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false
	}
	if role == RoleSuperUser {
		return true
	}
	for _, id := range appIDs {
		if id == appID {
			return true
		}
	}
	return false
}
