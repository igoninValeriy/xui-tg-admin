package permissions

import (
	"github.com/sirupsen/logrus"
)

// AccessType represents the access level of a user
type AccessType int

const (
	// None represents no access
	None AccessType = iota
	// Admin represents admin access
	Admin
)

// PermissionController manages user permissions
type PermissionController struct {
	adminIDs map[int64]bool
	logger   *logrus.Logger
}

// NewController creates a new permission controller
func NewController(adminIDs []int64, logger *logrus.Logger) *PermissionController {
	// Create a map for O(1) lookup of admin IDs
	adminIDMap := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		adminIDMap[id] = true
	}

	logger.Infof("Initialized permission controller with %d admins", len(adminIDs))

	return &PermissionController{
		adminIDs: adminIDMap,
		logger:   logger,
	}
}

// GetAccessType determines the access type of a user
func (p *PermissionController) GetAccessType(userID int64) AccessType {
	if p.IsAdmin(userID) {
		return Admin
	}

	// All other users have no access
	return None
}

// IsAdmin checks if a user is an admin
func (p *PermissionController) IsAdmin(userID int64) bool {
	isAdmin := p.adminIDs[userID]
	p.logger.Debugf("Checking if user %d is admin: %v", userID, isAdmin)
	return isAdmin
}
