package auth_test

import (
	"context"
	"testing"

	tests "github.com/MamangRust/microservice-point-of-sale-test"
	"github.com/stretchr/testify/suite"
)

type AuthGraphqlTestSuite struct {
	tests.BaseTestSuite
	gql         *tests.GraphQLClient
	email       string
	password    string
	userID      int
	accessToken string
}

func (s *AuthGraphqlTestSuite) SetupSuite() {
	s.BaseTestSuite.SetupSuite()

	// SetupAuthService starts the role + auth gRPC services. registerUser
	// assigns ROLE_ADMIN, which SetupTestSuite seeds (SeedMinimalRoles).
	s.SetupAuthService()

	s.email = "graphql.auth@example.com"
	s.password = "password123"

	s.gql = tests.NewGraphQLClient(s.T(), s.Conns, s.RedisClient(), s.Log)
}

// verifyUser simulates the email-verification step (login only accepts
// is_verified = true users).
func (s *AuthGraphqlTestSuite) verifyUser(email string) {
	_, err := s.SQLxDB().ExecContext(context.Background(),
		"UPDATE users SET is_verified = true WHERE email = $1", email)
	s.Require().NoError(err)
}

func (s *AuthGraphqlTestSuite) register(input map[string]interface{}) map[string]interface{} {
	res := s.gql.Do(`
		mutation Register($input: RegisterInput!) {
			registerUser(input: $input) { status message data { id email } }
		}`, map[string]interface{}{"input": input})
	return tests.Payload(s.T(), res, "registerUser")
}

func (s *AuthGraphqlTestSuite) Test1_Register() {
	created := s.register(map[string]interface{}{
		"firstname":        "Graphql",
		"lastname":         "Auth",
		"email":            s.email,
		"password":         s.password,
		"confirm_password": s.password,
	})
	s.userID = tests.Int(s.T(), created, "id")
	s.Require().NotZero(s.userID)
	s.Equal(s.email, tests.Str(s.T(), created, "email"))
}

func (s *AuthGraphqlTestSuite) Test2_Login() {
	s.verifyUser(s.email)

	login := s.gql.Do(`
		mutation Login($input: LoginInput!) {
			loginUser(input: $input) { status message data { access_token refresh_token } }
		}`, map[string]interface{}{"input": map[string]interface{}{
		"email":    s.email,
		"password": s.password,
	}})
	payload := tests.Payload(s.T(), login, "loginUser")
	s.accessToken = tests.Str(s.T(), payload, "access_token")
	s.Require().NotEmpty(s.accessToken)
}

func (s *AuthGraphqlTestSuite) Test3_GetMe() {
	s.Require().NotZero(s.userID)
	s.Require().NotEmpty(s.accessToken)

	// GetMe reads the caller id and the bearer token from the request context,
	// which AuthMiddleware normally populates.
	me := s.gql.WithUserAndToken(s.userID, s.accessToken).Do(`
		query Me($input: GetMeInput!) { getMe(input: $input) { status message data { id email } } }`,
		map[string]interface{}{"input": map[string]interface{}{"access_token": s.accessToken}})
	s.Equal(s.email, tests.Str(s.T(), tests.Payload(s.T(), me, "getMe"), "email"))
}

func (s *AuthGraphqlTestSuite) Test4_LoginLockout() {
	email := "graphql.locked@example.com"
	correct := "correctpassword"

	s.register(map[string]interface{}{
		"firstname":        "Graphql",
		"lastname":         "Locked",
		"email":            email,
		"password":         correct,
		"confirm_password": correct,
	})
	s.verifyUser(email)

	login := `mutation Login($input: LoginInput!) { loginUser(input: $input) { status } }`
	attempt := func(password string) []tests.GraphQLError {
		_, errs := s.gql.DoRaw(login, map[string]interface{}{"input": map[string]interface{}{
			"email":    email,
			"password": password,
		}})
		return errs
	}

	// The lockout threshold is 5 failed attempts.
	for i := 0; i < 5; i++ {
		s.NotEmpty(attempt("wrongpassword"), "attempt %d should fail", i+1)
	}

	// Once locked, even the correct password is rejected.
	s.NotEmpty(attempt(correct), "locked account should reject the correct password")
}

func TestAuthGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(AuthGraphqlTestSuite))
}
