package response

// RoleResponsePayload represents the base payload structure for role validation
// responses. It is exchanged over Kafka between the role service and the gateway.
type RoleResponsePayload struct {
	CorrelationID string   `json:"correlation_id"`
	Valid         bool     `json:"valid"`
	RoleNames     []string `json:"role_names"`
}

type RoleResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RoleResponseDeleteAt struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at"`
}

type ApiResponseRoleAll struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ApiResponseRoleDelete struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ApiResponseRole struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    *RoleResponse `json:"data"`
}

type ApiResponsesRole struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    []*RoleResponse `json:"data"`
}

type ApiResponsePaginationRole struct {
	Status     string          `json:"status"`
	Message    string          `json:"message"`
	Data       []*RoleResponse `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}

type ApiResponsePaginationRoleDeleteAt struct {
	Status     string                  `json:"status"`
	Message    string                  `json:"message"`
	Data       []*RoleResponseDeleteAt `json:"data"`
	Pagination *PaginationMeta         `json:"pagination"`
}
