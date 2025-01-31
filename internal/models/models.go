package models

type AdminCreds struct {
	PDSJWTSecret     string `json:"PDS_JWT_SECRET"`
	PDSAdminPassword string `json:"PDS_ADMIN_PASSWORD"`
	PDSAdminUsername string `json:"PDS_ADMIN_USERNAME"`
}

type EmailCreds struct {
	APIKey string `json:"RESEND_APIKEY"`
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
	DID        string `json:"did"`
	AccessJWT  string `json:"accessJwt"`
	RefreshJWT string `json:"refreshJwt"`
}

type UtilACcountCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DID      string `json:"did"`
}

type SessionRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type SessionResponse struct {
	AccessJwt string `json:"accessJwt"`
	Did       string `json:"did"`
	Handle    string `json:"handle"`
}

type BlockedUsernames struct {
	Generic []string `json:"generic"`
}
