package v1

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}

type AdminUserItem struct {
	UserID    string   `json:"userId"`
	Username  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type GetAdminUsersRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"pageSize" binding:"required,min=1,max=100"`
	Username string `form:"username"`
}

type GetAdminUsersResponseData struct {
	List  []AdminUserItem `json:"list"`
	Total int64           `json:"total"`
}

type AdminUserCreateRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Nickname string   `json:"nickname"`
	Password string   `json:"password" binding:"required,min=6"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
}

type AdminUserUpdateRequest struct {
	UserID   string   `json:"userId" binding:"required"`
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Nickname string   `json:"nickname"`
	Password string   `json:"password" binding:"omitempty,min=6"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
}

type AdminUserDeleteRequest struct {
	UserID string `form:"userId" binding:"required"`
}

type RoleItem struct {
	Sid  string `json:"sid"`
	Name string `json:"name"`
}

type RoleCreateRequest struct {
	Sid  string `json:"sid" binding:"required,min=2,max=50"`
	Name string `json:"name" binding:"required"`
}

type RoleUpdateRequest struct {
	Sid  string `json:"sid" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type RoleDeleteRequest struct {
	Sid string `form:"sid" binding:"required"`
}

type UpdateRolePermissionsRequest struct {
	Sid         string       `json:"sid" binding:"required"`
	Permissions []Permission `json:"permissions" binding:"required,dive"`
}

type Permission struct {
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type GetRolePermissionsResponseData struct {
	Sid         string       `json:"sid"`
	Permissions []Permission `json:"permissions"`
}
