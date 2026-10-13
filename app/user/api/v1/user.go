package v1

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"a@b.com"`
	Password string `json:"password" binding:"required,min=6" example:"123456"`
	Nickname string `json:"nickname" example:"alan"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"a@b.com"`
	Password string `json:"password" binding:"required" example:"123456"`
}

type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required" example:"alan"`
	Email    string `json:"email" binding:"required,email" example:"a@b.com"`
}

type ProfileResponseData struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}
