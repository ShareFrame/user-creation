package models

type AdminCreds struct {
	PDSJWTSecret     string `json:"PDS_JWT_SECRET"`
	PDSAdminPassword string `json:"PDS_ADMIN_PASSWORD"`
}

type UserRequest struct {
	Handle     string `json:"handle"`
	Email      string `json:"email"`
	InviteCode string `json:"code"`
}
