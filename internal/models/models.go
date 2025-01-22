package models

type AdminCreds struct {
	PDSJWTSecret     string `json:"PDS_JWT_SECRET"`
	PDSAdminPassword string `json:"PDS_ADMIN_PASSWORD"`
	PDSAdminUsername string `json:"PDS_ADMIN_USERNAME"`
}

type UserRequest struct {
	Handle string `json:"handle"`
	Email  string `json:"email"`
}

type InviteCodeResponse struct {
	Code string `json:"code"`
}

type CreateUserResponse struct {
	Handle     string `json:"handle"`
	Email      string `json:"email"`
	DID        string `json:"did"`
	AccessJWT  string `json:"accessJwt"`
	RefreshJWT string `json:"refreshJwt"`
}
